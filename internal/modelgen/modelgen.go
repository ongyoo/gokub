package modelgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Options struct {
	Root    string
	Name    string
	Package string
	Input   string
	Output  string
	Force   bool
}

type schema struct {
	Type       any               `json:"type"`
	Format     string            `json:"format"`
	Properties map[string]schema `json:"properties"`
	Required   []string          `json:"required"`
	Items      *schema           `json:"items"`
}

type definition struct {
	Name   string
	Fields []field
}

type field struct {
	Name     string
	JSONName string
	Type     string
	Optional bool
}

func Generate(options Options) (string, error) {
	if !validIdentifierBase(options.Name) {
		return "", fmt.Errorf("model name %q must contain letters, numbers, hyphens, or underscores", options.Name)
	}
	if options.Package == "" {
		options.Package = packageName(options.Name)
	}
	if !validPackage(options.Package) {
		return "", fmt.Errorf("invalid package name %q", options.Package)
	}
	input, err := os.ReadFile(options.Input)
	if err != nil {
		return "", fmt.Errorf("read JSON input: %w", err)
	}
	definitions, usesTime, err := parse(input, exported(options.Name))
	if err != nil {
		return "", err
	}
	source, err := render(options.Package, definitions, usesTime)
	if err != nil {
		return "", err
	}
	output := options.Output
	if output == "" {
		base := filepath.Join(options.Root, "internal")
		if info, err := os.Stat(filepath.Join(base, "domain")); err == nil && info.IsDir() {
			base = filepath.Join(base, "domain")
		}
		output = filepath.Join(base, options.Package, "model_gen.go")
	} else if !filepath.IsAbs(output) {
		output = filepath.Join(options.Root, output)
	}
	inside, err := pathInside(options.Root, output)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", fmt.Errorf("output path must be inside the project root")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}
	flags := os.O_WRONLY | os.O_CREATE
	if options.Force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(output, flags, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%s already exists; use --force to replace it", output)
		}
		return "", err
	}
	if _, err := file.Write(source); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return output, nil
}

func pathInside(root, path string) (bool, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	targetPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	relative, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false, err
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}

func parse(input []byte, rootName string) ([]definition, bool, error) {
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, false, fmt.Errorf("decode JSON: %w", err)
	}
	builder := &typeBuilder{}
	if object, ok := raw.(map[string]any); ok && (object["$schema"] != nil || object["properties"] != nil && object["type"] == "object") {
		var root schema
		if err := json.Unmarshal(input, &root); err != nil {
			return nil, false, fmt.Errorf("decode JSON Schema: %w", err)
		}
		rootType, _ := schemaType(root.Type)
		if rootType != "object" && len(root.Properties) == 0 {
			return nil, false, fmt.Errorf("top-level JSON Schema must describe an object")
		}
		builder.fromSchema(rootName, root)
	} else {
		object, ok := raw.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("top-level JSON value must be an object")
		}
		builder.fromSample(rootName, object)
	}
	return builder.definitions, builder.usesTime, nil
}

type typeBuilder struct {
	definitions []definition
	usesTime    bool
}

func (b *typeBuilder) fromSample(name string, object map[string]any) string {
	fields := make([]field, 0, len(object))
	for _, key := range sortedKeys(object) {
		fields = append(fields, field{Name: exported(key), JSONName: key, Type: b.sampleType(name+exported(key), object[key])})
	}
	b.definitions = append(b.definitions, definition{Name: name, Fields: fields})
	return name
}

func (b *typeBuilder) sampleType(name string, value any) string {
	switch typed := value.(type) {
	case nil:
		return "any"
	case bool:
		return "bool"
	case string:
		if _, err := time.Parse(time.RFC3339, typed); err == nil {
			b.usesTime = true
			return "time.Time"
		}
		return "string"
	case json.Number:
		if strings.ContainsAny(typed.String(), ".eE") {
			return "float64"
		}
		return "int64"
	case map[string]any:
		return b.fromSample(name, typed)
	case []any:
		for _, item := range typed {
			if item != nil {
				return "[]" + b.sampleType(singular(name), item)
			}
		}
		return "[]any"
	default:
		return "any"
	}
}

func (b *typeBuilder) fromSchema(name string, value schema) string {
	typeName, nullable := schemaType(value.Type)
	var result string
	switch typeName {
	case "object", "":
		required := stringSet(value.Required)
		fields := make([]field, 0, len(value.Properties))
		keys := make([]string, 0, len(value.Properties))
		for key := range value.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			property := value.Properties[key]
			fieldType := b.fromSchema(name+exported(key), property)
			optional := !required[key]
			if optional && !strings.HasPrefix(fieldType, "[]") && fieldType != "any" && !strings.HasPrefix(fieldType, "*") {
				fieldType = "*" + fieldType
			}
			fields = append(fields, field{Name: exported(key), JSONName: key, Type: fieldType, Optional: optional})
		}
		b.definitions = append(b.definitions, definition{Name: name, Fields: fields})
		result = name
	case "array":
		if value.Items == nil {
			result = "[]any"
		} else {
			result = "[]" + b.fromSchema(singular(name), *value.Items)
		}
	case "string":
		if value.Format == "date-time" {
			b.usesTime = true
			result = "time.Time"
		} else {
			result = "string"
		}
	case "integer":
		result = "int64"
	case "number":
		result = "float64"
	case "boolean":
		result = "bool"
	default:
		result = "any"
	}
	if nullable && result != "any" && !strings.HasPrefix(result, "[]") && !strings.HasPrefix(result, "*") {
		result = "*" + result
	}
	return result
}

func render(packageName string, definitions []definition, usesTime bool) ([]byte, error) {
	var output strings.Builder
	fmt.Fprintf(&output, "// Code generated by GOKUB from JSON. DO NOT EDIT.\npackage %s\n", packageName)
	if usesTime {
		output.WriteString("\nimport \"time\"\n")
	}
	for index := len(definitions) - 1; index >= 0; index-- {
		definition := definitions[index]
		fmt.Fprintf(&output, "\ntype %s struct {\n", definition.Name)
		for _, field := range definition.Fields {
			tag := field.JSONName
			if field.Optional {
				tag += ",omitempty"
			}
			fmt.Fprintf(&output, "\t%s %s `json:%s`\n", field.Name, field.Type, strconv.Quote(tag))
		}
		output.WriteString("}\n")
	}
	formatted, err := format.Source([]byte(output.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated model: %w", err)
	}
	return formatted, nil
}

func schemaType(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, typed == "null"
	case []any:
		selected := ""
		nullable := false
		for _, item := range typed {
			name, _ := item.(string)
			if name == "null" {
				nullable = true
			} else if selected == "" {
				selected = name
			}
		}
		return selected, nullable
	default:
		return "", false
	}
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func exported(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if len(parts) == 0 {
		return "Field"
	}
	for index, part := range parts {
		upper := strings.ToUpper(part)
		switch upper {
		case "ID", "URL", "HTTP", "HTTPS", "API", "IP", "UUID":
			parts[index] = upper
		default:
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[index] = string(runes)
		}
	}
	name := strings.Join(parts, "")
	if unicode.IsDigit([]rune(name)[0]) {
		name = "Field" + name
	}
	return name
}

func packageName(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(value, "-", ""), "_", ""))
}

func singular(value string) string {
	if strings.HasSuffix(value, "ies") {
		return strings.TrimSuffix(value, "ies") + "y"
	}
	if strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "ss") {
		return strings.TrimSuffix(value, "s")
	}
	return value + "Item"
}

func validIdentifierBase(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func validPackage(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if !(unicode.IsLower(r) || r == '_' || index > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}
