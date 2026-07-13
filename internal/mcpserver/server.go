package mcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gokub "github.com/ongyoo/gokub"
	"github.com/ongyoo/gokub/internal/agentskills"
	"github.com/ongyoo/gokub/internal/catalog"
	"github.com/ongyoo/gokub/internal/doctor"
	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/modelgen"
	"github.com/ongyoo/gokub/internal/plugins"
	"github.com/ongyoo/gokub/internal/projectgraph"
	"github.com/ongyoo/gokub/internal/projectstatus"
	"github.com/ongyoo/gokub/internal/projectupgrade"
	customtemplates "github.com/ongyoo/gokub/internal/templates"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func Serve(root string, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := encoder.Encode(response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		res := handle(root, req)
		if err := encoder.Encode(res); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handle(root string, req request) response {
	res := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		res.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]string{"name": "gokub", "version": gokub.Version},
		}
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = map[string]any{"tools": tools()}
	case "tools/call":
		var call toolCall
		if err := json.Unmarshal(req.Params, &call); err != nil {
			res.Error = &rpcError{Code: -32602, Message: "invalid tool arguments"}
			return res
		}
		text, err := callTool(root, call)
		res.Result = toolResult(text, err)
	default:
		res.Error = &rpcError{Code: -32601, Message: "method not found"}
	}
	return res
}

func tools() []tool {
	object := func(properties map[string]any, required ...string) map[string]any {
		if properties == nil {
			properties = map[string]any{}
		}
		schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	return []tool{
		{Name: "gokub_project_status", Description: "Read the current GOKUB project manifest.", InputSchema: object(nil)},
		{Name: "gokub_doctor", Description: "Check project structure, manifest, and generated files.", InputSchema: object(nil)},
		{Name: "gokub_project_score", Description: "Score project architecture, security, testing, and operations with recommendations.", InputSchema: object(nil)},
		{Name: "gokub_dependency_graph", Description: "Map internal Go package dependencies and architecture layers.", InputSchema: object(map[string]any{
			"include_tests": map[string]any{"type": "boolean", "description": "Include dependencies imported by Go test files."},
			"check":         map[string]any{"type": "boolean", "description": "Include cycle and architecture boundary analysis."},
		})},
		{Name: "gokub_project_upgrade", Description: "Check or apply a versioned GOKUB project metadata migration with backup.", InputSchema: object(map[string]any{
			"apply": map[string]any{"type": "boolean", "description": "Apply the planned migration. Defaults to check only."},
		})},
		{Name: "gokub_catalog", Description: "List available features, capabilities, providers, and recipes.", InputSchema: object(nil)},
		{Name: "gokub_plugins", Description: "List installed GOKUB plugins and their declared commands without executing them.", InputSchema: object(nil)},
		{Name: "gokub_add_feature", Description: "Generate a supported feature and update .gokub.yaml.", InputSchema: object(map[string]any{
			"feature": map[string]any{"type": "string", "enum": catalog.FeatureNames()},
			"name":    map[string]any{"type": "string", "description": "Required module name when feature is crud."},
		}, "feature")},
		{Name: "gokub_generate_model", Description: "Generate Go structs from a JSON sample or JSON Schema file.", InputSchema: object(map[string]any{
			"name":    map[string]any{"type": "string", "description": "Root Go model name."},
			"input":   map[string]any{"type": "string", "description": "JSON file path relative to the project root."},
			"package": map[string]any{"type": "string", "description": "Optional generated Go package name."},
			"force":   map[string]any{"type": "boolean", "description": "Replace an existing generated model."},
		}, "name", "input")},
		{Name: "gokub_install_template", Description: "Install a community project template from an HTTPS GitHub repository.", InputSchema: object(map[string]any{
			"repository": map[string]any{"type": "string", "description": "GitHub owner/repository or HTTPS URL."},
			"ref":        map[string]any{"type": "string", "description": "Optional branch or tag."},
			"subdir":     map[string]any{"type": "string", "description": "Optional template subdirectory."},
			"name":       map[string]any{"type": "string", "description": "Optional local template name."},
		}, "repository")},
		{Name: "gokub_search_templates", Description: "Search public GitHub repositories tagged gokub-template without installing them.", InputSchema: object(map[string]any{
			"query": map[string]any{"type": "string", "description": "Optional search terms."},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 50},
		})},
		{Name: "gokub_apply_recipe", Description: "Apply a supported feature recipe and update .gokub.yaml.", InputSchema: object(map[string]any{
			"name": map[string]any{"type": "string", "enum": catalog.RecipeNames()},
		}, "name")},
		{Name: "gokub_install_skills", Description: "Install the GOKUB project skill for coding agents.", InputSchema: object(map[string]any{
			"agent": map[string]any{"type": "string", "enum": []string{"all", "codex", "claude", "copilot", "gemini", "portable"}},
			"force": map[string]any{"type": "boolean", "description": "Replace existing skill files."},
		})},
	}
}

func callTool(root string, call toolCall) (string, error) {
	switch call.Name {
	case "gokub_project_status":
		report, err := projectstatus.Build(root)
		if err != nil {
			return "", err
		}
		content, err := json.MarshalIndent(report, "", "  ")
		return string(content), err
	case "gokub_doctor":
		content, err := json.MarshalIndent(doctor.Analyze(root), "", "  ")
		return string(content), err
	case "gokub_project_score":
		content, err := json.MarshalIndent(doctor.Score(root), "", "  ")
		return string(content), err
	case "gokub_dependency_graph":
		includeTests, _ := call.Arguments["include_tests"].(bool)
		graph, err := projectgraph.Build(root, includeTests)
		if err != nil {
			return "", err
		}
		value := any(graph)
		if check, _ := call.Arguments["check"].(bool); check {
			value = projectgraph.Analyze(graph)
		}
		content, err := json.MarshalIndent(value, "", "  ")
		return string(content), err
	case "gokub_project_upgrade":
		apply, _ := call.Arguments["apply"].(bool)
		var value any
		var err error
		if apply {
			value, err = projectupgrade.Apply(root, gokub.Version)
		} else {
			value, err = projectupgrade.Check(root, gokub.Version)
		}
		if err != nil {
			return "", err
		}
		content, err := json.MarshalIndent(value, "", "  ")
		return string(content), err
	case "gokub_catalog":
		content, err := json.MarshalIndent(map[string]any{
			"features": catalog.Features, "capabilities": catalog.Capabilities, "recipes": catalog.Recipes,
		}, "", "  ")
		return string(content), err
	case "gokub_plugins":
		items, err := plugins.List()
		if err != nil {
			return "", err
		}
		content, err := json.MarshalIndent(items, "", "  ")
		return string(content), err
	case "gokub_add_feature":
		feature := stringArgument(call.Arguments, "feature")
		name := stringArgument(call.Arguments, "name")
		if !catalog.HasFeature(feature) {
			return "", fmt.Errorf("unknown feature %q", feature)
		}
		if feature == "crud" && strings.TrimSpace(name) == "" {
			return "", fmt.Errorf("name is required for crud")
		}
		m, err := manifest.Read(filepath.Join(root, manifest.FileName))
		if err != nil {
			return "", err
		}
		if err := generator.AddFeature(root, feature, name); err != nil {
			return "", err
		}
		record := feature
		if feature == "crud" {
			record += ":" + name
		}
		manifest.AddFeature(&m, record)
		if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
			return "", err
		}
		return "added " + record, nil
	case "gokub_generate_model":
		name := stringArgument(call.Arguments, "name")
		input := stringArgument(call.Arguments, "input")
		packageName := stringArgument(call.Arguments, "package")
		force, _ := call.Arguments["force"].(bool)
		cleanInput := filepath.Clean(input)
		if filepath.IsAbs(input) || cleanInput == ".." || strings.HasPrefix(cleanInput, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("input must be a path inside the project")
		}
		inputPath := filepath.Join(root, cleanInput)
		resolvedRoot, rootErr := filepath.EvalSymlinks(root)
		resolvedInput, inputErr := filepath.EvalSymlinks(inputPath)
		if rootErr != nil || inputErr != nil {
			return "", fmt.Errorf("resolve input path inside project")
		}
		relativeInput, err := filepath.Rel(resolvedRoot, resolvedInput)
		if err != nil || relativeInput == ".." || strings.HasPrefix(relativeInput, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("input must be a path inside the project")
		}
		path, err := modelgen.Generate(modelgen.Options{
			Root: root, Name: name, Package: packageName, Input: resolvedInput, Force: force,
		})
		if err != nil {
			return "", err
		}
		relative, _ := filepath.Rel(root, path)
		return "generated " + filepath.ToSlash(relative), nil
	case "gokub_install_template":
		installed, err := customtemplates.Install(customtemplates.InstallOptions{
			Repository: stringArgument(call.Arguments, "repository"),
			Ref:        stringArgument(call.Arguments, "ref"),
			Subdir:     stringArgument(call.Arguments, "subdir"),
			Name:       stringArgument(call.Arguments, "name"),
		})
		if err != nil {
			return "", err
		}
		return "installed community template " + installed, nil
	case "gokub_search_templates":
		limit := 10
		if value, ok := call.Arguments["limit"].(float64); ok {
			limit = int(value)
		}
		items, err := customtemplates.Search(customtemplates.SearchOptions{
			Query: stringArgument(call.Arguments, "query"), Limit: limit, Token: os.Getenv("GITHUB_TOKEN"),
		})
		if err != nil {
			return "", err
		}
		content, err := json.MarshalIndent(items, "", "  ")
		return string(content), err
	case "gokub_apply_recipe":
		name := stringArgument(call.Arguments, "name")
		recipe, ok := catalog.Recipes[name]
		if !ok {
			return "", fmt.Errorf("unknown recipe %q", name)
		}
		m, err := manifest.Read(filepath.Join(root, manifest.FileName))
		if err != nil {
			return "", err
		}
		for _, feature := range recipe.Features {
			if err := generator.AddFeature(root, feature, ""); err != nil {
				return "", err
			}
			manifest.AddFeature(&m, feature)
		}
		manifest.AddRecipe(&m, recipe.Name)
		if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
			return "", err
		}
		return "applied recipe " + recipe.Name, nil
	case "gokub_install_skills":
		agent := stringArgument(call.Arguments, "agent")
		if agent == "" {
			agent = "all"
		}
		force, _ := call.Arguments["force"].(bool)
		written, err := agentskills.Install(root, agent, force)
		if err != nil {
			return "", err
		}
		if len(written) == 0 {
			return "GOKUB skill pack is already installed", nil
		}
		return fmt.Sprintf("installed GOKUB skill pack for %s (%d files)", agent, len(written)), nil
	default:
		return "", fmt.Errorf("unknown tool %q", call.Name)
	}
}

func toolResult(text string, err error) map[string]any {
	if err != nil {
		return map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true}
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": text}}, "isError": false}
}

func stringArgument(arguments map[string]any, name string) string {
	value, _ := arguments[name].(string)
	return strings.TrimSpace(value)
}
