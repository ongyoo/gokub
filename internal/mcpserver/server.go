package mcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	gokub "github.com/gokub/gokub"
	"github.com/gokub/gokub/internal/agentskills"
	"github.com/gokub/gokub/internal/catalog"
	"github.com/gokub/gokub/internal/doctor"
	"github.com/gokub/gokub/internal/generator"
	"github.com/gokub/gokub/internal/manifest"
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
		{Name: "gokub_catalog", Description: "List available features, capabilities, providers, and recipes.", InputSchema: object(nil)},
		{Name: "gokub_add_feature", Description: "Generate a supported feature and update .gokub.yaml.", InputSchema: object(map[string]any{
			"feature": map[string]any{"type": "string", "enum": catalog.FeatureNames()},
			"name":    map[string]any{"type": "string", "description": "Required module name when feature is crud."},
		}, "feature")},
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
		m, err := manifest.Read(filepath.Join(root, manifest.FileName))
		if err != nil {
			return "", fmt.Errorf("read project manifest: %w", err)
		}
		content, err := json.MarshalIndent(m, "", "  ")
		return string(content), err
	case "gokub_doctor":
		results := doctor.Check(root)
		content, err := json.MarshalIndent(results, "", "  ")
		return string(content), err
	case "gokub_catalog":
		content, err := json.MarshalIndent(map[string]any{
			"features": catalog.Features, "capabilities": catalog.Capabilities, "recipes": catalog.Recipes,
		}, "", "  ")
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
