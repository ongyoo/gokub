package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gokub "github.com/gokub/gokub"
	"github.com/gokub/gokub/internal/agentskills"
	"github.com/gokub/gokub/internal/catalog"
	"github.com/gokub/gokub/internal/doctor"
	"github.com/gokub/gokub/internal/generator"
	"github.com/gokub/gokub/internal/manifest"
	"github.com/gokub/gokub/internal/mcpserver"
	customtemplates "github.com/gokub/gokub/internal/templates"
)

func Run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) > 0 && args[0] == "mcp" {
		return runMCP(args[1:], in, out)
	}
	startupLogo(out, gokub.Version)
	if len(args) == 0 {
		usage(out)
		return nil
	}
	switch args[0] {
	case "new":
		return runNew(args[1:], in, out)
	case "add":
		return runAdd(args[1:], out)
	case "remove":
		return runRemove(args[1:], out)
	case "enable":
		return runEnable(args[1:], in, out)
	case "disable":
		return runDisable(args[1:], out)
	case "switch":
		return runSwitch(args[1:], out)
	case "status":
		return runStatus(out)
	case "doctor":
		return runDoctor(out)
	case "update":
		return runUpdate(args[1:], out)
	case "recipe":
		return runRecipe(args[1:], out)
	case "agent":
		return runAgent(args[1:], out)
	case "template":
		return runTemplate(args[1:], out)
	case "skill", "skills":
		return runSkill(args[1:], out)
	case "uninstall":
		return runUninstall(args[1:], in, out)
	case "version", "--version", "-v":
		runVersion(out)
		return nil
	case "help", "--help", "-h":
		help(args[1:], out)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSkill(args []string, out io.Writer) error {
	operation := "list"
	if len(args) > 0 {
		operation = args[0]
		args = args[1:]
	}
	if operation == "list" {
		if len(args) != 0 {
			return fmt.Errorf("usage: gokub skill list")
		}
		section(out, "Agent Skills")
		for _, name := range []string{"gokub-project", "gokub-add-domain", "gokub-verify-change"} {
			fmt.Fprintf(out, "  %s\n", name)
		}
		status := agentskills.Status(".")
		p := newPalette()
		for _, agent := range []string{"codex", "claude", "copilot", "gemini", "portable"} {
			state := p.dim("not installed")
			if status[agent] {
				state = p.ok("installed")
			}
			fmt.Fprintf(out, "  %-10s %s\n", agent, state)
		}
		return nil
	}
	fs := flag.NewFlagSet("skill "+operation, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	agent := fs.String("agent", "all", "agent target")
	force := fs.Bool("force", false, "replace existing skill files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub skill %s [--agent all|codex|claude|copilot|gemini|portable]", operation)
	}
	switch operation {
	case "install":
		written, err := agentskills.Install(".", *agent, *force)
		if err != nil {
			return err
		}
		if len(written) == 0 {
			fmt.Fprintln(out, newPalette().dim("Skills are already installed. Use --force to refresh them."))
			return nil
		}
		success(out, "installed GOKUB skill pack for %s", *agent)
		for _, path := range written {
			fmt.Fprintf(out, "  %s\n", newPalette().dim(path))
		}
		return nil
	case "remove":
		removed, err := agentskills.Remove(".", *agent)
		if err != nil {
			return err
		}
		if len(removed) == 0 {
			fmt.Fprintln(out, newPalette().dim("No installed skill files found."))
			return nil
		}
		success(out, "removed GOKUB skill pack for %s", *agent)
		return nil
	default:
		return fmt.Errorf("usage: gokub skill [list|install|remove]")
	}
}

func runUninstall(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "skip confirmation")
	purge := fs.Bool("purge", false, "remove custom templates and local GOKUB data")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub uninstall [--yes] [--purge]")
	}
	if !*yes {
		prompt := newPrompter(in, out, 1)
		if prompt.choice("Uninstall GOKUB CLI?", []string{"No", "Yes"}, "No") != "Yes" {
			fmt.Fprintln(out, newPalette().dim("Uninstall cancelled."))
			return nil
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate GOKUB executable: %w", err)
	}
	if err := removeExecutable(executable); err != nil {
		return err
	}
	success(out, "removed %s", executable)
	if *purge {
		if err := customtemplates.Purge(); err != nil {
			return fmt.Errorf("CLI removed, but local data could not be purged: %w", err)
		}
		success(out, "removed local GOKUB templates and configuration")
	} else {
		fmt.Fprintln(out, newPalette().dim("Custom templates in ~/.gokub were kept. Use --purge to remove them."))
	}
	return nil
}

func removeExecutable(path string) error {
	base := strings.ToLower(filepath.Base(path))
	if base != "gokub" && base != "gokub.exe" {
		return fmt.Errorf("refusing to remove unexpected executable %q", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

func runTemplate(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "list" {
		names, err := customtemplates.Names()
		if err != nil {
			return err
		}
		section(out, "Custom Templates")
		if len(names) == 0 {
			fmt.Fprintln(out, newPalette().dim("  No custom templates installed."))
			return nil
		}
		for _, name := range names {
			commandLine(out, name, "installed")
		}
		return nil
	}
	switch args[0] {
	case "add":
		if len(args) < 2 || len(args) > 3 {
			return fmt.Errorf("usage: gokub template add [name] <path>")
		}
		name, path := "", args[1]
		if len(args) == 3 {
			name, path = args[1], args[2]
		}
		installed, err := customtemplates.Add(name, path)
		if err != nil {
			return err
		}
		success(out, "installed template %s", installed)
		return nil
	case "remove":
		if len(args) != 2 {
			return fmt.Errorf("usage: gokub template remove <name>")
		}
		if err := customtemplates.Remove(args[1]); err != nil {
			return err
		}
		success(out, "removed template %s", args[1])
		return nil
	default:
		return fmt.Errorf("usage: gokub template [list|add [name] <path>|remove <name>]")
	}
}

func runMCP(args []string, in io.Reader, out io.Writer) error {
	if len(args) != 1 || args[0] != "serve" {
		return fmt.Errorf("usage: gokub mcp serve")
	}
	return mcpserver.Serve(".", in, out)
}

func runAgent(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "help" {
		help([]string{"agent"}, out)
		return nil
	}
	if args[0] != "init" {
		return fmt.Errorf("usage: gokub agent init [--provider codex|claude|copilot|gemini|portable|all]")
	}
	fs := flag.NewFlagSet("agent init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "all", "agent provider")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	written, err := generator.WriteAgentFiles(".", *provider)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	for _, file := range written {
		success(out, "wrote %s", file)
	}
	return nil
}

func runVersion(out io.Writer) {
	p := newPalette()
	fmt.Fprintln(out, p.cyan("Version Details"))
	fmt.Fprintf(out, "  %s %s\n", p.dim("cli         "), p.silver("gokub"))
	fmt.Fprintf(out, "  %s %s\n", p.dim("version     "), p.amber(gokub.Version))
	fmt.Fprintf(out, "  %s %s\n", p.dim("kit         "), p.silver("Go Project Kit"))
	fmt.Fprintf(out, "  %s %s\n", p.dim("update repo "), p.silver(repository()))
	fmt.Fprintf(out, "  %s %s\n", p.dim("update check"), p.amber("gokub update --check"))
}

func runNew(args []string, in io.Reader, out io.Writer) error {
	positionalName := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positionalName = args[0]
		args = args[1:]
	}
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	template := fs.String("template", "gin-clean", "project template")
	style := fs.String("style", "monolith", "project style")
	framework := fs.String("framework", "gin", "web framework")
	database := fs.String("database", "postgres", "database")
	architecture := fs.String("architecture", "clean", "architecture")
	messaging := fs.String("messaging", "none", "messaging provider")
	recipe := fs.String("recipe", "", "recipe to apply")
	module := fs.String("module", "", "Go module path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	setFlags := visitedFlags(fs)
	wizard := positionalName == "" && fs.NArg() == 0
	prompts := newPrompter(in, out, newWizardTotal(wizard, fs.NArg(), positionalName, *module, setFlags))

	name := ""
	if positionalName != "" {
		name = positionalName
	} else if fs.NArg() > 0 {
		name = fs.Arg(0)
	} else {
		wizardHeader(out)
		name = prompts.ask("Project name", "example-api")
	}
	if wizard && *module == "" {
		*module = prompts.ask("Go module", "github.com/example/"+name)
	}
	if wizard && !setFlags["style"] {
		*style = prompts.choice("Project style", []string{"monolith", "microservices"}, *style)
	}
	if wizard && !setFlags["template"] {
		if *style == "microservices" {
			*template = "microservices"
		} else {
			*template = "monolith"
		}
		templateOptions := []string{"monolith", "microservices", "gin-clean", "fiber-clean", "worker", "grpc-service"}
		if names, err := customtemplates.Names(); err == nil {
			templateOptions = append(templateOptions, names...)
		}
		*template = prompts.choice("Template", templateOptions, *template)
	}
	if wizard && !setFlags["framework"] {
		*framework = prompts.choice("Framework", []string{"gin", "fiber", "grpc", "none"}, *framework)
	}
	if wizard && !setFlags["database"] {
		*database = prompts.choice("Database", []string{"postgres", "mongodb", "none"}, *database)
	}
	if wizard && !setFlags["architecture"] {
		*architecture = prompts.choice("Architecture", []string{"clean", "hexagonal", "layered"}, *architecture)
	}
	if wizard && !setFlags["messaging"] {
		*messaging = prompts.choice("Messaging", []string{"none", "kafka", "rabbitmq", "nats"}, *messaging)
	}
	if wizard && !setFlags["recipe"] {
		*recipe = prompts.choice("Recipe", append([]string{"none"}, catalog.RecipeNames()...), "none")
		if *recipe == "none" {
			*recipe = ""
		}
	}
	if !wizard && setFlags["style"] && !setFlags["template"] {
		if *style == "microservices" {
			*template = "microservices"
		} else {
			*template = "monolith"
		}
	}
	m := manifest.New(name, *module)
	m.Template = *template
	m.Style = *style
	m.Framework = *framework
	m.Database = *database
	m.Architecture = *architecture
	m.Messaging = *messaging
	if err := validateProjectOptions(m); err != nil {
		return err
	}
	if m.Database != "none" {
		manifest.AddFeature(&m, m.Database)
	}
	if m.Messaging != "none" {
		manifest.AddFeature(&m, m.Messaging)
	}
	if wizard {
		renderProjectSummary(out, m, *recipe)
	}

	if *recipe != "" {
		r, ok := catalog.Recipes[*recipe]
		if !ok {
			return fmt.Errorf("unknown recipe %q", *recipe)
		}
		manifest.AddRecipe(&m, r.Name)
		for _, feature := range r.Features {
			manifest.AddFeature(&m, feature)
		}
	}

	if wizard {
		p := newPalette()
		fmt.Fprintln(out, p.dim("Generating project files..."))
	}
	if err := generator.NewProject(".", m); err != nil {
		return err
	}
	for _, feature := range m.Features {
		if (m.Template == "monolith" || m.Template == "microservices") && standardTemplateFeature(feature) {
			continue
		}
		if err := generator.AddFeature(name, feature, ""); err != nil {
			return err
		}
	}
	success(out, "created %s", name)
	p := newPalette()
	fmt.Fprintf(out, "  %s\n  %s\n", p.dim("next:"), p.amber("cd "+name+" && go test ./..."))
	return nil
}

func standardTemplateFeature(feature string) bool {
	return contains([]string{
		"auth", "redis", "postgres", "mongodb", "kafka", "rabbitmq", "nats",
		"otel", "docker", "github-actions",
	}, feature)
}

func validateProjectOptions(m manifest.Manifest) error {
	if err := manifest.Validate(m); err != nil {
		return err
	}
	allowed := map[string][]string{
		"style":        {"monolith", "microservices"},
		"framework":    {"gin", "fiber", "grpc", "none"},
		"database":     {"postgres", "mongodb", "none"},
		"architecture": {"clean", "hexagonal", "layered"},
		"messaging":    {"none", "kafka", "rabbitmq", "nats"},
	}
	values := map[string]string{
		"style": m.Style, "framework": m.Framework, "database": m.Database,
		"architecture": m.Architecture, "messaging": m.Messaging,
	}
	for field, options := range allowed {
		if !contains(options, values[field]) {
			return fmt.Errorf("unknown %s %q (choose: %s)", field, values[field], strings.Join(options, ", "))
		}
	}
	builtIn := contains([]string{"monolith", "microservices", "gin-clean", "fiber-clean", "worker", "grpc-service"}, m.Template)
	_, custom, err := customtemplates.Resolve(m.Template)
	if err != nil {
		return err
	}
	if !builtIn && !custom {
		return fmt.Errorf("unknown template %q; install it with gokub template add <path>", m.Template)
	}
	return nil
}

func runAdd(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub add <feature> [name]")
	}
	feature := args[0]
	if !catalog.HasFeature(feature) {
		return fmt.Errorf("unknown feature %q", feature)
	}
	name := ""
	if len(args) > 1 {
		name = args[1]
	}
	m, err := manifest.Read(manifest.FileName)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	if err := generator.AddFeature(".", feature, name); err != nil {
		return err
	}
	record := feature
	if feature == "crud" && name != "" {
		record = "crud:" + name
	}
	manifest.AddFeature(&m, record)
	if err := manifest.Write(manifest.FileName, m); err != nil {
		return err
	}
	success(out, "added %s", record)
	return nil
}

func runRemove(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub remove <feature>")
	}
	m, err := manifest.Read(manifest.FileName)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	manifest.RemoveFeature(&m, args[0])
	if m.Database == args[0] {
		m.Database = "none"
	}
	if m.Messaging == args[0] {
		m.Messaging = "none"
	}
	if err := manifest.Write(manifest.FileName, m); err != nil {
		return err
	}
	success(out, "removed %s from manifest", args[0])
	return nil
}

func runEnable(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub enable <capability> [provider]")
	}
	capabilityName := args[0]
	capability, ok := catalog.Capabilities[capabilityName]
	if !ok {
		return fmt.Errorf("unknown capability %q", capabilityName)
	}
	provider := ""
	if len(args) > 1 {
		provider = args[1]
		if !catalog.ProviderForCapability(capabilityName, provider) {
			return fmt.Errorf("%q is not a provider for capability %q", provider, capabilityName)
		}
	} else {
		prompts := newPrompter(in, out, 1)
		provider = prompts.choice("Provider for "+capabilityName, capability.Providers, capability.Providers[0])
	}
	return applyCapabilityProvider(".", capabilityName, provider, false, out)
}

func runSwitch(args []string, out io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gokub switch <capability> <provider>")
	}
	capabilityName := args[0]
	provider := args[1]
	if !catalog.ProviderForCapability(capabilityName, provider) {
		return fmt.Errorf("%q is not a provider for capability %q", provider, capabilityName)
	}
	return applyCapabilityProvider(".", capabilityName, provider, true, out)
}

func runDisable(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub disable <capability> [provider]")
	}
	capabilityName := args[0]
	capability, ok := catalog.Capabilities[capabilityName]
	if !ok {
		return fmt.Errorf("unknown capability %q", capabilityName)
	}
	remove := capability.Providers
	if len(args) > 1 {
		provider := args[1]
		if !catalog.ProviderForCapability(capabilityName, provider) {
			return fmt.Errorf("%q is not a provider for capability %q", provider, capabilityName)
		}
		remove = []string{provider}
	}
	m, err := manifest.Read(manifest.FileName)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	manifest.RemoveFeatures(&m, remove)
	switch capabilityName {
	case "messaging":
		if len(args) == 1 || contains(remove, m.Messaging) {
			m.Messaging = "none"
		}
	case "database":
		if len(args) == 1 || contains(remove, m.Database) {
			m.Database = "none"
		}
	}
	if err := manifest.Write(manifest.FileName, m); err != nil {
		return err
	}
	success(out, "disabled %s", capabilityName)
	return nil
}

func runStatus(out io.Writer) error {
	m, err := manifest.Read(manifest.FileName)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	p := newPalette()
	section(out, "Project")
	fmt.Fprintf(out, "  %s %s\n", p.dim("name        "), p.silver(m.Name))
	fmt.Fprintf(out, "  %s %s\n", p.dim("module      "), p.silver(m.Module))
	fmt.Fprintf(out, "  %s %s\n", p.dim("template    "), p.amber(m.Template))
	fmt.Fprintf(out, "  %s %s\n", p.dim("style       "), p.silver(m.Style))
	fmt.Fprintf(out, "  %s %s\n", p.dim("architecture"), p.silver(m.Architecture))
	fmt.Fprintln(out)
	section(out, "Capabilities")
	for _, name := range catalog.CapabilityNames() {
		capability := catalog.Capabilities[name]
		enabled := enabledProvidersForCapability(m, capability.Name)
		if len(enabled) == 0 {
			fmt.Fprintf(out, "  [%s] %-16s %s\n", p.dim("off"), capability.Name, p.dim(strings.Join(capability.Providers, ", ")))
			continue
		}
		fmt.Fprintf(out, "  [%s] %-16s %s\n", p.ok("on"), capability.Name, p.amber(strings.Join(enabled, ", ")))
	}
	if len(m.Recipes) > 0 {
		fmt.Fprintln(out)
		section(out, "Recipes")
		for _, recipe := range m.Recipes {
			fmt.Fprintf(out, "  %s\n", p.silver(recipe))
		}
	}
	return nil
}

func applyCapabilityProvider(root, capabilityName, provider string, replace bool, out io.Writer) error {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	capability := catalog.Capabilities[capabilityName]
	if replace {
		manifest.RemoveFeatures(&m, capability.Providers)
	}
	if err := generator.AddFeature(root, provider, ""); err != nil {
		return err
	}
	manifest.AddFeature(&m, provider)
	switch capabilityName {
	case "messaging":
		m.Messaging = provider
	case "database":
		m.Database = provider
	}
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		return err
	}
	action := "enabled"
	if replace {
		action = "switched"
	}
	success(out, "%s %s with %s", action, capabilityName, provider)
	return nil
}

func enabledProviders(m manifest.Manifest, providers []string) []string {
	enabled := []string{}
	for _, provider := range providers {
		if contains(m.Features, provider) {
			enabled = append(enabled, provider)
		}
	}
	return enabled
}

func enabledProvidersForCapability(m manifest.Manifest, capabilityName string) []string {
	capability := catalog.Capabilities[capabilityName]
	enabled := enabledProviders(m, capability.Providers)
	switch capabilityName {
	case "database":
		if m.Database != "" && m.Database != "none" && catalog.ProviderForCapability(capabilityName, m.Database) && !contains(enabled, m.Database) {
			enabled = append(enabled, m.Database)
		}
	case "messaging":
		if m.Messaging != "" && m.Messaging != "none" && catalog.ProviderForCapability(capabilityName, m.Messaging) && !contains(enabled, m.Messaging) {
			enabled = append(enabled, m.Messaging)
		}
	}
	return enabled
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func runRecipe(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "list" {
		section(out, "Recipes")
		for _, name := range catalog.RecipeNames() {
			recipe := catalog.Recipes[name]
			commandLine(out, recipe.Name, recipe.Description)
		}
		return nil
	}
	if args[0] != "add" || len(args) < 2 {
		return fmt.Errorf("usage: gokub recipe add <name>")
	}
	recipe, ok := catalog.Recipes[args[1]]
	if !ok {
		return fmt.Errorf("unknown recipe %q", args[1])
	}
	m, err := manifest.Read(manifest.FileName)
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	for _, feature := range recipe.Features {
		if err := generator.AddFeature(".", feature, ""); err != nil {
			return err
		}
		manifest.AddFeature(&m, feature)
	}
	manifest.AddRecipe(&m, recipe.Name)
	if err := manifest.Write(manifest.FileName, m); err != nil {
		return err
	}
	success(out, "applied recipe %s", recipe.Name)
	return nil
}

func runDoctor(out io.Writer) error {
	results := doctor.Check(".")
	failed := 0
	p := newPalette()
	section(out, "Doctor")
	for _, result := range results {
		state := p.ok("ok")
		if !result.OK {
			state = p.fail("fail")
			failed++
		}
		fmt.Fprintf(out, "  [%s] %-20s %s\n", state, result.Name, p.dim(result.Info))
	}
	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	return nil
}

func runUpdate(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	repo := fs.String("repo", repository(), "GitHub repository")
	checkOnly := fs.Bool("check", true, "check latest release")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*checkOnly {
		return fmt.Errorf("self-update install is not enabled yet; use --check")
	}
	latest, err := latestRelease(*repo)
	if err != nil {
		return err
	}
	p := newPalette()
	fmt.Fprintf(out, "%s %s\n%s  %s\n", p.dim("current"), p.silver(gokub.Version), p.dim("latest "), p.amber(latest))
	return nil
}

func repository() string {
	if value := os.Getenv("GOKUB_REPOSITORY"); value != "" {
		return value
	}
	return gokub.Repository
}

func latestRelease(repo string) (string, error) {
	client := http.Client{Timeout: 5 * time.Second}
	url := "https://api.github.com/repos/" + repo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub release check failed: %s", resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("latest release response did not include tag_name")
	}
	return payload.TagName, nil
}

type prompter struct {
	reader *bufio.Reader
	input  *os.File
	out    io.Writer
	step   int
	total  int
}

func newPrompter(in io.Reader, out io.Writer, total int) *prompter {
	input, _ := in.(*os.File)
	return &prompter{reader: bufio.NewReader(in), input: input, out: out, total: total}
}

func (p *prompter) ask(label, fallback string) string {
	step := p.nextStep()
	return promptStepReader(p.reader, p.out, step, p.total, label, fallback)
}

func (p *prompter) choice(label string, options []string, fallback string) string {
	step := p.nextStep()
	if p.input != nil && terminalAvailable(p.input) {
		if value, ok := p.choiceInteractive(step, label, options, fallback); ok {
			return value
		}
	}

	pal := newPalette()
	fmt.Fprintf(p.out, "\n%s %s\n", stepBadge(pal, step, p.total), pal.silver(label))
	for i, option := range options {
		marker := " "
		if option == fallback {
			marker = "recommended"
		}
		fmt.Fprintf(p.out, "  %d. %-14s %s\n", i+1, pal.amber(option), pal.dim(marker))
	}
	value := promptReader(p.reader, p.out, "Select "+label, fallback)
	if index, err := strconv.Atoi(value); err == nil && index >= 1 && index <= len(options) {
		return options[index-1]
	}
	for _, option := range options {
		if strings.EqualFold(value, option) {
			return option
		}
	}
	return fallback
}

func (p *prompter) choiceInteractive(step int, label string, options []string, fallback string) (string, bool) {
	selected := selectedIndex(options, fallback)
	lines := renderChoice(p.out, step, p.total, label, options, fallback, selected)
	for {
		key, ok, err := readImmediateKey(p.input)
		if err != nil || !ok {
			return "", false
		}
		switch key {
		case keyEnter:
			clearLines(p.out, lines)
			renderSelected(p.out, step, p.total, label, options[selected])
			return options[selected], true
		case keyCtrlC:
			clearLines(p.out, lines)
			renderSelected(p.out, step, p.total, label, fallback)
			return fallback, true
		case keyUp:
			if selected == 0 {
				selected = len(options) - 1
			} else {
				selected--
			}
		case keyDown:
			selected = (selected + 1) % len(options)
		default:
			if key >= '1' && key <= '9' {
				index := int(key - '1')
				if index >= 0 && index < len(options) {
					selected = index
					clearLines(p.out, lines)
					renderSelected(p.out, step, p.total, label, options[selected])
					return options[selected], true
				}
			} else {
				fmt.Fprint(p.out, "\a")
			}
		}
		clearLines(p.out, lines)
		lines = renderChoice(p.out, step, p.total, label, options, fallback, selected)
	}
}

func renderChoice(out io.Writer, step, total int, label string, options []string, fallback string, selected int) int {
	pal := newPalette()
	fmt.Fprintf(out, "\n%s %s %s\n", stepBadge(pal, step, total), pal.silver(label), pal.dim("default: "+fallback))
	for i, option := range options {
		cursor := "  "
		value := " " + option + " "
		if i == selected {
			cursor = pal.cyan(">")
			value = pal.selected(value)
		} else {
			value = pal.silver(value)
		}
		marker := ""
		if option == fallback {
			marker = pal.dim(" recommended")
		}
		fmt.Fprintf(out, "  %s  %s%s\n", cursor, value, marker)
	}
	fmt.Fprintf(out, "%s\n", pal.dim("  Up/Down to move, Enter to select. Number keys still work."))
	return len(options) + 3
}

func renderSelected(out io.Writer, step, total int, label, value string) {
	pal := newPalette()
	fmt.Fprintf(out, "%s %s %s\n", stepBadge(pal, step, total), pal.silver(label+":"), pal.amber(value))
}

func clearLines(out io.Writer, lines int) {
	fmt.Fprintf(out, "\x1b[%dA\x1b[J", lines)
}

func (p *prompter) nextStep() int {
	p.step++
	return p.step
}

func stepBadge(p palette, step, total int) string {
	return p.cyan(fmt.Sprintf("[%d/%d]", step, total))
}

func selectedIndex(options []string, fallback string) int {
	for i, option := range options {
		if option == fallback {
			return i
		}
	}
	return 0
}

func newWizardTotal(wizard bool, argCount int, positionalName string, module string, setFlags map[string]bool) int {
	if !wizard {
		return 0
	}
	total := 0
	if positionalName == "" && argCount == 0 {
		total++
	}
	if module == "" {
		total++
	}
	for _, name := range []string{"style", "template", "framework", "database", "architecture", "messaging", "recipe"} {
		if !setFlags[name] {
			total++
		}
	}
	return total
}

func renderProjectSummary(out io.Writer, m manifest.Manifest, recipe string) {
	pal := newPalette()
	if recipe == "" {
		recipe = "none"
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, pal.cyan("Project Profile"))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("name        "), pal.silver(m.Name))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("module      "), pal.silver(m.Module))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("template    "), pal.amber(m.Template))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("style       "), pal.silver(m.Style))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("framework   "), pal.silver(m.Framework))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("database    "), pal.silver(m.Database))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("architecture"), pal.silver(m.Architecture))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("messaging   "), pal.silver(m.Messaging))
	fmt.Fprintf(out, "  %s %s\n\n", pal.dim("recipe      "), pal.amber(recipe))
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func wizardHeader(out io.Writer) {
	p := newPalette()
	fmt.Fprintln(out, p.cyan("GOKUB Project Wizard"))
	fmt.Fprintln(out, p.dim("Use Enter for recommended values. Use Up/Down in menus."))
	fmt.Fprintln(out)
}

func prompt(in io.Reader, out io.Writer, label, fallback string) string {
	return promptReader(bufio.NewReader(in), out, label, fallback)
}

func promptReader(reader *bufio.Reader, out io.Writer, label, fallback string) string {
	return promptStepReader(reader, out, 0, 0, label, fallback)
}

func promptStepReader(reader *bufio.Reader, out io.Writer, step, total int, label, fallback string) string {
	p := newPalette()
	if step > 0 && total > 0 {
		fmt.Fprintf(out, "%s %s %s\n", stepBadge(p, step, total), p.silver(label), p.dim("default: "+fallback))
		fmt.Fprintf(out, "%s ", p.cyan(">"))
	} else {
		fmt.Fprintf(out, "%s %s %s ", p.cyan("?"), p.silver(label), p.dim("["+fallback+"]"))
	}
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func usage(out io.Writer) {
	exe := filepath.Base(os.Args[0])
	p := newPalette()
	banner(out)
	section(out, "Commands")
	commandLine(out, exe+" new", "start the step-by-step project wizard")
	commandLine(out, exe+" new [name] [--recipe event-driven]", "create from flags for scripts")
	commandLine(out, exe+" add <feature> [name]", "add a capability or CRUD module")
	commandLine(out, exe+" remove <feature>", "remove a capability from the manifest")
	commandLine(out, exe+" enable <capability> [provider]", "enable a capability with provider selection")
	commandLine(out, exe+" disable <capability> [provider]", "disable a capability or provider")
	commandLine(out, exe+" switch <capability> <provider>", "replace a capability provider")
	commandLine(out, exe+" status", "show project and capability state")
	commandLine(out, exe+" doctor", "check project health")
	commandLine(out, exe+" update --check", "check GitHub release updates")
	commandLine(out, exe+" recipe [list|add <name>]", "manage capability bundles")
	commandLine(out, exe+" template [list|add|remove]", "manage custom project templates")
	commandLine(out, exe+" skill [list|install|remove]", "manage Codex, Claude, Copilot, and agent skills")
	commandLine(out, exe+" agent init [--provider <agent>]", "create AI agent guidance and MCP files")
	commandLine(out, exe+" mcp serve", "expose project tools to Codex, Claude, and MCP agents")
	commandLine(out, exe+" uninstall", "remove the installed GOKUB CLI")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n\n", p.dim("Details:"), p.amber(exe+" help <command>"))
}

func help(args []string, out io.Writer) {
	if len(args) == 0 {
		usage(out)
		return
	}
	p := newPalette()
	topic := args[0]
	section(out, strings.ToUpper(topic[:1])+topic[1:])
	switch args[0] {
	case "new":
		commandLine(out, "gokub new", "start the step-by-step project wizard")
		commandLine(out, "gokub new [name] [flags]", "create from flags for scripts")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Flags"))
		commandLine(out, "--module <path>", "Go module path")
		commandLine(out, "--template <name>", "project template, default gin-clean")
		commandLine(out, "--style <name>", "project style: monolith or microservices")
		commandLine(out, "--framework <name>", "web framework, default gin")
		commandLine(out, "--database <name>", "database provider, default postgres")
		commandLine(out, "--architecture <name>", "architecture style, default clean")
		commandLine(out, "--messaging <name>", "messaging provider, default none")
		commandLine(out, "--recipe <name>", "apply a recipe during creation")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub new", "")
		commandLine(out, "gokub new payment-api --module github.com/acme/payment-api", "")
		commandLine(out, "gokub new payment-api --recipe event-driven", "")
	case "add":
		commandLine(out, "gokub add <feature> [name]", "add a capability or CRUD module")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Features"))
		for _, name := range catalog.FeatureNames() {
			feature := catalog.Features[name]
			commandLine(out, feature.Name, feature.Description)
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub add auth", "")
		commandLine(out, "gokub add crud product", "")
		commandLine(out, "gokub add kafka", "")
	case "remove":
		commandLine(out, "gokub remove <feature>", "remove a feature from .gokub.yaml")
		fmt.Fprintln(out, p.dim("Generated files are kept for review."))
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Example"))
		commandLine(out, "gokub remove kafka", "")
	case "enable":
		commandLine(out, "gokub enable <capability> [provider]", "enable a capability with provider selection")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Capabilities"))
		for _, name := range catalog.CapabilityNames() {
			capability := catalog.Capabilities[name]
			commandLine(out, capability.Name, capability.Description+" ["+strings.Join(capability.Providers, ", ")+"]")
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub enable messaging", "")
		commandLine(out, "gokub enable messaging kafka", "")
		commandLine(out, "gokub enable authentication", "")
	case "disable":
		commandLine(out, "gokub disable <capability> [provider]", "disable a capability or provider")
		fmt.Fprintln(out, p.dim("Generated files are kept for review."))
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub disable messaging", "")
		commandLine(out, "gokub disable messaging kafka", "")
		commandLine(out, "gokub disable cache", "")
	case "switch":
		commandLine(out, "gokub switch <capability> <provider>", "replace a capability provider")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub switch messaging rabbitmq", "")
		commandLine(out, "gokub switch messaging kafka", "")
	case "status":
		commandLine(out, "gokub status", "show project and capability state")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Example"))
		commandLine(out, "gokub status", "")
	case "doctor":
		commandLine(out, "gokub doctor", "check project structure, manifest readability, and generated folders")
	case "update":
		commandLine(out, "gokub update --check [--repo owner/repo]", "check latest GitHub release tag")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Example"))
		commandLine(out, "gokub update --check --repo gokub/gokub", "")
	case "recipe":
		commandLine(out, "gokub recipe list", "list capability bundles")
		commandLine(out, "gokub recipe add <name>", "install multiple capabilities")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Recipes"))
		for _, name := range catalog.RecipeNames() {
			recipe := catalog.Recipes[name]
			commandLine(out, recipe.Name, recipe.Description)
		}
	case "agent":
		commandLine(out, "gokub agent init [--provider <agent>]", "create AI agent guidance and skills")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Files"))
		commandLine(out, "codex", "AGENTS.md and .codex/config.toml")
		commandLine(out, "claude", "CLAUDE.md and .mcp.json")
		commandLine(out, "copilot", ".github skills and repository instructions")
		commandLine(out, "gemini", "portable skills and GEMINI.md")
		commandLine(out, "portable", "portable .agents/skills pack")
		commandLine(out, "all", "guidance and MCP configuration for both agents")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub agent init", "")
		commandLine(out, "gokub agent init --provider codex", "")
		commandLine(out, "gokub agent init --provider claude", "")
	case "template":
		commandLine(out, "gokub template add <path>", "install a folder using its directory name")
		commandLine(out, "gokub template add <name> <path>", "install a folder with a custom name")
		commandLine(out, "gokub template list", "list installed custom templates")
		commandLine(out, "gokub template remove <name>", "remove an installed custom template")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Template placeholders"))
		for _, placeholder := range []string{"{{project_name}}", "{{module}}", "{{template}}", "{{style}}", "{{framework}}", "{{database}}", "{{architecture}}", "{{messaging}}"} {
			commandLine(out, placeholder, "")
		}
	case "skill", "skills":
		commandLine(out, "gokub skill install", "install gokub-project for all supported agents")
		commandLine(out, "gokub skill install --agent codex", "install the portable Codex project skill")
		commandLine(out, "gokub skill install --agent claude", "install the Claude project skill")
		commandLine(out, "gokub skill install --agent copilot", "install the GitHub Copilot project skill")
		commandLine(out, "gokub skill install --agent gemini", "install portable skill and Gemini guidance")
		commandLine(out, "gokub skill list", "show skill installation status")
		commandLine(out, "gokub skill remove --agent all", "remove installed skill directories")
		commandLine(out, "--force", "replace existing skill and instruction files")
	case "mcp":
		commandLine(out, "gokub mcp serve", "start the GOKUB MCP server over stdio")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Generated agent configuration"))
		commandLine(out, ".codex/config.toml", "Codex project MCP configuration")
		commandLine(out, ".mcp.json", "Claude and compatible MCP client configuration")
	case "uninstall":
		commandLine(out, "gokub uninstall", "confirm and remove the current CLI executable")
		commandLine(out, "gokub uninstall --yes", "uninstall without an interactive prompt")
		commandLine(out, "gokub uninstall --purge", "also remove custom templates and ~/.gokub data")
	default:
		fmt.Fprintf(out, "unknown help topic %q\n\n", args[0])
		usage(out)
	}
	fmt.Fprintln(out)
}
