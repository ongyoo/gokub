package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	gokub "github.com/ongyoo/gokub"
	"github.com/ongyoo/gokub/internal/agentskills"
	"github.com/ongyoo/gokub/internal/catalog"
	"github.com/ongyoo/gokub/internal/completion"
	"github.com/ongyoo/gokub/internal/doctor"
	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/goversion"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/mcpserver"
	"github.com/ongyoo/gokub/internal/modelgen"
	"github.com/ongyoo/gokub/internal/plugins"
	"github.com/ongyoo/gokub/internal/projectgraph"
	"github.com/ongyoo/gokub/internal/projectstatus"
	"github.com/ongyoo/gokub/internal/projectupgrade"
	"github.com/ongyoo/gokub/internal/selfupdate"
	customtemplates "github.com/ongyoo/gokub/internal/templates"
)

func Run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) > 0 && args[0] == "mcp" {
		return runMCP(args[1:], in, out)
	}
	if !machineOutputMode(args) {
		startupLogo(out, gokub.Version)
	}
	if len(args) == 0 {
		return runCommandCenter(in, out, errOut)
	}
	switch args[0] {
	case "new":
		return runNew(args[1:], in, out)
	case "add":
		return runAdd(args[1:], in, out)
	case "remove":
		return runRemove(args[1:], out)
	case "enable":
		return runEnable(args[1:], in, out)
	case "disable":
		return runDisable(args[1:], out)
	case "switch":
		return runSwitch(args[1:], out)
	case "status":
		return runStatus(args[1:], out)
	case "doctor":
		return runDoctor(args[1:], out)
	case "score":
		return runScore(args[1:], out)
	case "graph":
		return runGraph(args[1:], out)
	case "upgrade":
		return runUpgrade(args[1:], in, out)
	case "update":
		return runUpdate(args[1:], in, out)
	case "recipe":
		return runRecipe(args[1:], out)
	case "agent":
		return runAgent(args[1:], out)
	case "template":
		return runTemplate(args[1:], in, out)
	case "plugin":
		return runPlugin(args[1:], in, out, errOut)
	case "skill", "skills":
		return runSkill(args[1:], out)
	case "completion":
		return runCompletion(args[1:], out)
	case "uninstall":
		return runUninstall(args[1:], in, out)
	case "version":
		return runVersion(args[1:], out)
	case "--version", "-v":
		return runVersion(nil, out)
	case "help", "--help", "-h":
		help(args[1:], out)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runCommandCenter(in io.Reader, out, errOut io.Writer) error {
	input, ok := in.(*os.File)
	if !ok || !terminalAvailable(input) {
		usage(out)
		return nil
	}
	palette := newPalette()
	for {
		_, projectErr := os.Stat(manifest.FileName)
		inProject := projectErr == nil
		actions := commandCenterActions(inProject)
		selected, ok := newPrompter(in, out, 1).menuChoice("Choose workflow", actions, actions[0])
		if !ok {
			// Input ended (EOF or closed terminal): leave the command center.
			return nil
		}
		exit, err := runCommandCenterAction(selected, in, out, errOut)
		if err != nil {
			fmt.Fprintf(out, "  %s %v\n", palette.fail("error:"), err)
		}
		if exit {
			return nil
		}
		fmt.Fprintln(out)
	}
}

// runCommandCenterAction runs one menu selection. It returns exit=true when the
// command center should stop: after a project is created or a feature is added,
// or when the user chooses Exit. Read-only actions return exit=false so the menu
// keeps looping and stays ready for the next command.
func runCommandCenterAction(selected string, in io.Reader, out, errOut io.Writer) (bool, error) {
	switch selected {
	case "New project":
		return true, runNew(nil, in, out)
	case "Add feature":
		options := append([]string{"custom module"}, catalog.FeatureNames()...)
		feature := newPrompter(in, out, 1).choice("Feature", options, "custom module")
		name := ""
		switch feature {
		case "custom module":
			feature = "crud"
			name = newPrompter(in, out, 1).ask("Module name", "product")
		case "crud":
			name = newPrompter(in, out, 1).ask("Resource name", "product")
		}
		args := []string{feature}
		if name != "" {
			args = append(args, name)
		}
		return true, runAdd(args, in, out)
	case "Generate model from JSON":
		return true, runAdd([]string{"model"}, in, out)
	case "Project status":
		return false, runStatus(nil, out)
	case "Doctor":
		return false, runDoctor(nil, out)
	case "Project score":
		return false, runScore(nil, out)
	case "Dependency graph":
		return false, runGraph(nil, out)
	case "Upgrade project":
		return false, runUpgrade(nil, in, out)
	case "Community templates":
		operation := newPrompter(in, out, 1).choice("Template workflow", []string{"Search and install", "List installed", "Add local folder"}, "Search and install")
		switch operation {
		case "Search and install":
			query := newPrompter(in, out, 1).ask("Search", "api")
			return false, runTemplate([]string{"search", query, "--install"}, in, out)
		case "Add local folder":
			path := newPrompter(in, out, 1).ask("Template folder", "./template")
			return false, runTemplate([]string{"add", path}, in, out)
		default:
			return false, runTemplate([]string{"list"}, in, out)
		}
	case "Plugins":
		return false, runPlugin([]string{"list"}, in, out, errOut)
	case "Agent skills":
		return false, runSkill([]string{"list"}, out)
	case "Update CLI":
		return false, runUpdate(nil, in, out)
	case "Install shell completion":
		return false, runCompletion([]string{"install"}, out)
	case "Help":
		help(nil, out)
		return false, nil
	case "Exit":
		return true, nil
	default:
		usage(out)
		return true, nil
	}
}

func commandCenterActions(inProject bool) []string {
	if !inProject {
		return []string{"New project", "Community templates", "Plugins", "Install shell completion", "Update CLI", "Help", "Exit"}
	}
	return []string{
		"Add feature", "Generate model from JSON", "Project status", "Doctor",
		"Project score", "Dependency graph", "Upgrade project", "Community templates",
		"Plugins", "Agent skills", "Install shell completion", "Update CLI", "New project", "Help", "Exit",
	}
}

func runCompletion(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub completion <bash|zsh|fish|install>")
	}
	if args[0] == "install" {
		if len(args) > 2 {
			return fmt.Errorf("usage: gokub completion install [bash|zsh|fish]")
		}
		shell := ""
		if len(args) == 2 {
			shell = args[1]
		}
		path, err := completion.Install(shell)
		if err != nil {
			return err
		}
		success(out, "installed shell completion at %s", path)
		fmt.Fprintln(out, newPalette().dim("Open a new terminal session to activate it."))
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gokub completion <bash|zsh|fish>")
	}
	script, err := completion.Script(args[0])
	if err != nil {
		return err
	}
	fmt.Fprint(out, script)
	return nil
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
	if homebrewExecutable(executable) {
		return fmt.Errorf("GOKUB is managed by Homebrew; run `brew uninstall gokub` so Homebrew can remove it cleanly")
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

func homebrewExecutable(path string) bool {
	paths := []string{path}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		paths = append(paths, resolved)
	}
	for _, candidate := range paths {
		slashPath := filepath.ToSlash(candidate)
		if strings.Contains(slashPath, "/Cellar/gokub/") || strings.Contains(slashPath, "/linuxbrew/.linuxbrew/Cellar/gokub/") {
			return true
		}
	}
	return false
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

func runTemplate(args []string, in io.Reader, out io.Writer) error {
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
	case "search":
		query := ""
		flagArgs := args[1:]
		if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
			query = flagArgs[0]
			flagArgs = flagArgs[1:]
		}
		fs := flag.NewFlagSet("template search", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		limit := fs.Int("limit", 10, "maximum search results")
		install := fs.Bool("install", false, "choose and install a search result")
		if err := fs.Parse(flagArgs); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: gokub template search [query] [--limit 10] [--install]")
		}
		results, err := customtemplates.Search(customtemplates.SearchOptions{
			Query: query, Limit: *limit, Token: os.Getenv("GITHUB_TOKEN"),
		})
		if err != nil {
			return err
		}
		section(out, "Community Templates")
		if len(results) == 0 {
			fmt.Fprintln(out, newPalette().dim("  No repositories with the gokub-template topic matched."))
			return nil
		}
		for _, result := range results {
			detail := result.Description
			if detail == "" {
				detail = "No description"
			}
			commandLine(out, result.Repository, fmt.Sprintf("%d stars  %s", result.Stars, detail))
		}
		if !*install {
			fmt.Fprintf(out, "\n  %s\n", newPalette().dim("Install: gokub template install <owner/repo>"))
			return nil
		}
		choices := make([]string, 0, len(results))
		for _, result := range results {
			choices = append(choices, result.Repository)
		}
		selected := newPrompter(in, out, 1).choice("Install template", choices, choices[0])
		installed, err := customtemplates.Install(customtemplates.InstallOptions{Repository: selected})
		if err != nil {
			return err
		}
		success(out, "installed community template %s", installed)
		return nil
	case "install":
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return fmt.Errorf("usage: gokub template install <owner/repo|GitHub URL> [--ref tag] [--subdir path] [--name name]")
		}
		fs := flag.NewFlagSet("template install", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		ref := fs.String("ref", "", "Git branch or tag")
		subdir := fs.String("subdir", "", "template directory inside the repository")
		name := fs.String("name", "", "installed template name")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: gokub template install <owner/repo|GitHub URL> [--ref tag] [--subdir path] [--name name]")
		}
		installed, err := customtemplates.Install(customtemplates.InstallOptions{
			Repository: args[1], Ref: *ref, Subdir: *subdir, Name: *name,
		})
		if err != nil {
			return err
		}
		success(out, "installed community template %s", installed)
		return nil
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
		return fmt.Errorf("usage: gokub template [list|install|add|remove]")
	}
}

func runPlugin(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "list" {
		items, err := plugins.List()
		if err != nil {
			return err
		}
		section(out, "Plugins")
		if len(items) == 0 {
			fmt.Fprintln(out, newPalette().dim("  No plugins installed."))
			return nil
		}
		for _, item := range items {
			commandLine(out, item.Name+" "+item.Version, item.Description)
			for _, command := range item.Commands {
				fmt.Fprintf(out, "    %s %s\n", newPalette().amber(command.Name), newPalette().dim(command.Description))
			}
		}
		return nil
	}
	switch args[0] {
	case "create":
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return fmt.Errorf("usage: gokub plugin create <name> [--module module/path]")
		}
		fs := flag.NewFlagSet("plugin create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		module := fs.String("module", "", "Go module path")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: gokub plugin create <name> [--module module/path]")
		}
		path, err := plugins.Create(".", args[1], *module)
		if err != nil {
			return err
		}
		success(out, "created plugin scaffold %s", path)
		fmt.Fprintf(out, "  %s\n", newPalette().amber("cd "+path+" && make build && gokub plugin install ."))
		return nil
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: gokub plugin install <path>")
		}
		installed, err := plugins.Install(args[1])
		if err != nil {
			return err
		}
		success(out, "installed plugin %s %s", installed.Name, installed.Version)
		return nil
	case "pack":
		source := "."
		flagArgs := args[1:]
		if len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "-") {
			source = flagArgs[0]
			flagArgs = flagArgs[1:]
		}
		fs := flag.NewFlagSet("plugin pack", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		outputDir := fs.String("output", "", "artifact output directory")
		if err := fs.Parse(flagArgs); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: gokub plugin pack [path] [--output directory]")
		}
		artifact, err := plugins.Pack(source, *outputDir)
		if err != nil {
			return err
		}
		success(out, "packed plugin artifact")
		fmt.Fprintf(out, "  %s %s\n  %s %s\n  %s %s\n", newPalette().dim("archive "), artifact.Archive, newPalette().dim("checksum"), artifact.ChecksumFile, newPalette().dim("sha256  "), artifact.SHA256)
		return nil
	case "verify":
		if len(args) < 2 || len(args) > 3 {
			return fmt.Errorf("usage: gokub plugin verify <archive> [checksum-file]")
		}
		checksum := ""
		if len(args) == 3 {
			checksum = args[2]
		}
		artifact, err := plugins.Verify(args[1], checksum)
		if err != nil {
			return err
		}
		success(out, "verified plugin artifact %s", artifact.SHA256)
		return nil
	case "remove":
		if len(args) != 2 {
			return fmt.Errorf("usage: gokub plugin remove <name>")
		}
		if err := plugins.Remove(args[1]); err != nil {
			return err
		}
		success(out, "removed plugin %s", args[1])
		return nil
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("usage: gokub plugin run <name> [command] [args...]")
		}
		return plugins.Execute(".", args[1], args[2:], in, out, errOut)
	default:
		return fmt.Errorf("usage: gokub plugin [list|create|install|pack|verify|run|remove]")
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

type versionInfo struct {
	CLI        string `json:"cli"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	BuildDate  string `json:"build_date"`
	GoVersion  string `json:"go_version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

func runVersion(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write machine-readable build metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub version [--json]")
	}
	info := versionInfo{
		CLI: "gokub", Version: gokub.Version, Repository: repository(),
		Commit: gokub.Commit, BuildDate: gokub.BuildDate, GoVersion: runtime.Version(),
		OS: runtime.GOOS, Arch: runtime.GOARCH,
	}
	if *jsonOutput {
		return json.NewEncoder(out).Encode(info)
	}
	p := newPalette()
	fmt.Fprintln(out, p.cyan("Version Details"))
	fmt.Fprintf(out, "  %s %s\n", p.dim("cli         "), p.silver(info.CLI))
	fmt.Fprintf(out, "  %s %s\n", p.dim("version     "), p.amber(info.Version))
	fmt.Fprintf(out, "  %s %s\n", p.dim("kit         "), p.silver("Go Project Kit"))
	fmt.Fprintf(out, "  %s %s\n", p.dim("commit      "), p.silver(info.Commit))
	fmt.Fprintf(out, "  %s %s\n", p.dim("built       "), p.silver(info.BuildDate))
	fmt.Fprintf(out, "  %s %s/%s %s\n", p.dim("runtime     "), p.silver(info.OS), p.silver(info.Arch), p.dim(info.GoVersion))
	fmt.Fprintf(out, "  %s %s\n", p.dim("update repo "), p.silver(info.Repository))
	fmt.Fprintf(out, "  %s %s\n", p.dim("update check"), p.amber("gokub update --check"))
	return nil
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
	agents := fs.String("agents", "all", "AI coding assistants: all|codex|claude|copilot|gemini|none")
	recipe := fs.String("recipe", "", "recipe to apply")
	module := fs.String("module", "", "Go module path")
	goVersion := fs.String("go-version", goversion.Recommended, "Go language version (major.minor)")
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
	if wizard && !setFlags["go-version"] {
		selection := prompts.choice("Go version", []string{
			goversion.Recommended + " (recommended)",
			goversion.Conservative + " (conservative)",
			"Custom",
		}, goversion.Recommended+" (recommended)")
		switch selection {
		case "Custom":
			*goVersion = promptStepReader(prompts.reader, out, 0, 0, "Go version (major.minor)", goversion.Conservative)
		default:
			*goVersion = strings.Fields(selection)[0]
		}
	}
	// Only ask about a template when the user has installed community templates;
	// otherwise the built-in kit generator is used and the question is noise.
	if wizard && !setFlags["template"] {
		if names, err := customtemplates.Names(); err == nil && len(names) > 0 {
			templateOptions := append([]string{"gin-clean", "fiber-clean"}, names...)
			*template = prompts.choice("Template", templateOptions, *template)
		}
	}
	if wizard && !setFlags["framework"] {
		*framework = prompts.choice("Framework", []string{"gin", "fiber", "echo"}, *framework)
	}
	if wizard && !setFlags["database"] {
		*database = prompts.choice("Database", []string{"postgres", "mongodb", "none"}, *database)
	}
	if wizard && !setFlags["messaging"] {
		*messaging = prompts.choice("Messaging", []string{"none", "kafka", "rabbitmq", "nats"}, *messaging)
	}
	if wizard && !setFlags["agents"] {
		*agents = prompts.choice("Vibe coding assistants", []string{"all", "codex", "claude", "copilot", "gemini", "none"}, *agents)
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
	m.GoVersion = *goVersion
	m.Template = *template
	m.Style = *style
	m.Framework = *framework
	m.Database = *database
	m.Architecture = *architecture
	m.Messaging = *messaging
	m.Agents = *agents
	if err := validateProjectOptions(m); err != nil {
		return err
	}
	if support := goversion.Classify(m.GoVersion); support == goversion.Unsupported || support == goversion.Future {
		p := newPalette()
		fmt.Fprintf(out, "  %s Go %s: %s\n", p.amber("note:"), m.GoVersion, goversion.Description(m.GoVersion))
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

	var genFile *os.File
	if f, ok := out.(*os.File); ok {
		genFile = f
	}
	spin := startSpinner(out, genFile, "generating project…")
	err := generator.NewProject(".", m)
	spin.Stop()
	if err != nil {
		return err
	}
	for _, feature := range m.Features {
		// The kit generator already provides database, HTTP, messaging, and CI
		// wiring, so skip standalone infrastructure scaffolds during creation.
		if standardTemplateFeature(feature) {
			continue
		}
		if err := generator.AddFeature(name, feature, ""); err != nil {
			return err
		}
	}
	success(out, "created %s", name)
	p := newPalette()
	// On an interactive terminal, step into the new project and open the command
	// center so the next action is one keypress away. (A child process cannot
	// change the parent shell's directory, so this applies within GOKUB only.)
	if inFile, ok := in.(*os.File); ok && terminalAvailable(inFile) {
		if err := os.Chdir(name); err == nil {
			fmt.Fprintf(out, "  %s %s\n", p.dim("entering project"), p.amber(name))
			fmt.Fprintf(out, "  %s\n", p.dim("run 'cd "+name+"' in your shell to stay here after exit"))
			return runCommandCenter(in, out, out)
		}
	}
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
		"framework":    {"gin", "fiber", "echo"},
		"database":     {"postgres", "mongodb", "none"},
		"architecture": {"clean", "hexagonal", "layered"},
		"messaging":    {"none", "kafka", "rabbitmq", "nats"},
		"agents":       {"all", "codex", "claude", "copilot", "gemini", "none"},
	}
	values := map[string]string{
		"style": m.Style, "framework": m.Framework, "database": m.Database,
		"architecture": m.Architecture, "messaging": m.Messaging, "agents": m.Agents,
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

func runAdd(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gokub add <feature> [name]")
	}
	feature := args[0]
	if feature == "model" {
		return runAddModel(args[1:], in, out)
	}
	// `custom` generates a named domain module (the CRUD five-file scaffold).
	if feature == "custom" {
		if len(args) < 2 {
			return fmt.Errorf("usage: gokub add custom <name>")
		}
		feature = "crud"
	}
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
	if err := generator.TidyModule("."); err != nil {
		return fmt.Errorf("resolve dependencies: %w", err)
	}
	success(out, "added %s", record)
	return nil
}

func runAddModel(args []string, in io.Reader, out io.Writer) error {
	name := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		flagArgs = args[1:]
	}
	fs := flag.NewFlagSet("add model", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	input := fs.String("from", "", "JSON sample or JSON Schema path")
	packageName := fs.String("package", "", "generated Go package name")
	output := fs.String("output", "", "generated Go file path")
	force := fs.Bool("force", false, "replace an existing generated model")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub add model [name] [--from file.json] [--package name] [--output path] [--force]")
	}
	if _, err := manifest.Read(manifest.FileName); err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	promptTotal := 0
	if *input == "" {
		promptTotal++
	}
	if name == "" {
		promptTotal++
	}
	prompts := newPrompter(in, out, promptTotal)
	if *input == "" {
		files, err := discoverJSONFiles(".")
		if err != nil {
			return err
		}
		if len(files) == 0 {
			*input = prompts.ask("JSON file", "model.json")
		} else {
			*input = prompts.choice("JSON source", files, files[0])
		}
	}
	if name == "" {
		name = prompts.ask("Model name", modelNameFromPath(*input))
	}
	path, err := modelgen.Generate(modelgen.Options{
		Root: ".", Name: name, Package: *packageName, Input: *input, Output: *output, Force: *force,
	})
	if err != nil {
		return err
	}
	success(out, "generated %s from %s", path, *input)
	return nil
}

func discoverJSONFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || contains([]string{"vendor", "node_modules", "dist"}, entry.Name())) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".json") {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find JSON files: %w", err)
	}
	sort.SliceStable(files, func(i, j int) bool {
		iSchema := strings.HasSuffix(strings.ToLower(files[i]), ".schema.json")
		jSchema := strings.HasSuffix(strings.ToLower(files[j]), ".schema.json")
		if iSchema != jSchema {
			return iSchema
		}
		return files[i] < files[j]
	})
	return files, nil
}

func modelNameFromPath(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.TrimSuffix(name, ".schema")
	if name == "" {
		return "model"
	}
	return name
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
			generator.UnwireMessaging(".")
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

func runStatus(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write machine-readable project status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub status [--json]")
	}
	report, err := projectstatus.Build(".")
	if err != nil {
		return fmt.Errorf("run inside a GOKUB project: %w", err)
	}
	if *jsonOutput {
		return json.NewEncoder(out).Encode(report)
	}
	p := newPalette()
	section(out, "Project")
	fmt.Fprintf(out, "  %s %s\n", p.dim("name        "), p.silver(report.Project.Name))
	fmt.Fprintf(out, "  %s %s\n", p.dim("module      "), p.silver(report.Project.Module))
	fmt.Fprintf(out, "  %s %s\n", p.dim("template    "), p.amber(report.Project.Template))
	fmt.Fprintf(out, "  %s %s\n", p.dim("style       "), p.silver(report.Project.Style))
	fmt.Fprintf(out, "  %s %s %s\n", p.dim("go          "), p.amber(report.Project.GoVersion), p.dim(report.Project.GoSupport))
	fmt.Fprintf(out, "  %s %s\n", p.dim("architecture"), p.silver(report.Project.Architecture))
	fmt.Fprintf(out, "  %s %d\n", p.dim("schema      "), report.SchemaVersion)
	fmt.Fprintf(out, "  %s %s\n", p.dim("generator   "), p.silver(report.GeneratorVersion))
	fmt.Fprintf(out, "  %s %s\n", p.dim("go guidance "), p.dim(report.Project.GoGuidance))
	fmt.Fprintln(out)
	section(out, "Capabilities")
	for _, capability := range report.Capabilities {
		if !capability.Enabled {
			fmt.Fprintf(out, "  [%s] %-16s %s\n", p.dim("off"), capability.Name, p.dim(strings.Join(capability.AvailableProviders, ", ")))
			continue
		}
		fmt.Fprintf(out, "  [%s] %-16s %s\n", p.ok("on"), capability.Name, p.amber(strings.Join(capability.Providers, ", ")))
	}
	if len(report.Recipes) > 0 {
		fmt.Fprintln(out)
		section(out, "Recipes")
		for _, recipe := range report.Recipes {
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
	if capabilityName == "messaging" {
		if err := generator.TidyModule(root); err != nil {
			return fmt.Errorf("resolve dependencies: %w", err)
		}
	}
	action := "enabled"
	if replace {
		action = "switched"
	}
	success(out, "%s %s with %s", action, capabilityName, provider)
	return nil
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
	if err := generator.TidyModule("."); err != nil {
		return fmt.Errorf("resolve dependencies: %w", err)
	}
	success(out, "applied recipe %s", recipe.Name)
	return nil
}

func runDoctor(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write a machine-readable report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub doctor [--json]")
	}
	report := doctor.Analyze(".")
	if *jsonOutput {
		content, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(content))
		if !report.OK {
			return fmt.Errorf("%d checks failed", report.Failed)
		}
		return nil
	}
	p := newPalette()
	section(out, "Doctor")
	for _, result := range report.Checks {
		state := p.ok("ok")
		if !result.OK {
			state = p.fail("fail")
		}
		fmt.Fprintf(out, "  [%s] %-20s %s\n", state, result.Name, p.dim(result.Info))
	}
	if !report.OK {
		return fmt.Errorf("%d checks failed", report.Failed)
	}
	return nil
}

func runScore(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "write machine-readable JSON")
	failUnder := fs.Int("fail-under", 0, "fail when the score is below this value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub score [--json] [--fail-under 0-100]")
	}
	if *failUnder < 0 || *failUnder > 100 {
		return fmt.Errorf("--fail-under must be between 0 and 100")
	}
	report := doctor.Score(".")
	if *jsonOutput {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			return err
		}
		return scoreThresholdError(report.Score, *failUnder)
	}
	p := newPalette()
	section(out, "Project Score")
	fmt.Fprintf(out, "  %s %s  %s\n\n", p.amber(strconv.Itoa(report.Score)+"/"+strconv.Itoa(report.Max)), p.silver(report.Grade), scoreBar(report.Score, report.Max))
	for _, category := range report.Categories {
		fmt.Fprintf(out, "  %-14s %s %2d/%d\n", category.Name, scoreBar(category.Score, category.Max), category.Score, category.Max)
	}
	fmt.Fprintln(out)
	section(out, "Recommended Next Steps")
	recommendations := 0
	for _, category := range report.Categories {
		for _, check := range category.Checks {
			if check.OK {
				continue
			}
			recommendations++
			fmt.Fprintf(out, "  %s %-14s %s\n", p.amber("+"), p.dim(category.Name), p.silver(check.Recommendation))
		}
	}
	if recommendations == 0 {
		fmt.Fprintf(out, "  %s\n", p.ok("All score checks passed."))
	}
	return scoreThresholdError(report.Score, *failUnder)
}

func scoreThresholdError(score, threshold int) error {
	if threshold > 0 && score < threshold {
		return fmt.Errorf("project score %d is below required threshold %d", score, threshold)
	}
	return nil
}

func machineOutputMode(args []string) bool {
	if len(args) < 1 {
		return false
	}
	if args[0] == "completion" && len(args) == 2 && args[1] != "install" {
		return true
	}
	if args[0] == "version" && len(args) == 2 && args[1] == "--json" {
		return true
	}
	if args[0] == "status" && len(args) == 2 && args[1] == "--json" {
		return true
	}
	if len(args) < 2 {
		return false
	}
	if args[0] == "score" && contains(args[1:], "--json") {
		return true
	}
	if args[0] == "doctor" && contains(args[1:], "--json") {
		return true
	}
	if args[0] == "upgrade" && contains(args[1:], "--json") {
		return true
	}
	if args[0] == "update" && contains(args[1:], "--json") {
		return true
	}
	if args[0] != "graph" {
		return false
	}
	for index, arg := range args[1:] {
		if arg == "--format=json" || arg == "--format=mermaid" {
			return true
		}
		if arg == "--format" && index+2 < len(args) && (args[index+2] == "json" || args[index+2] == "mermaid") {
			return true
		}
	}
	return false
}

func scoreBar(score, max int) string {
	const width = 10
	filled := 0
	if max > 0 {
		filled = score * width / max
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "]"
}

func runGraph(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "output format: text, mermaid, or json")
	includeTests := fs.Bool("include-tests", false, "include dependencies from Go test files")
	check := fs.Bool("check", false, "fail on dependency cycles or architecture boundary violations")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || (*format != "text" && *format != "mermaid" && *format != "json") {
		return fmt.Errorf("usage: gokub graph [--format text|mermaid|json] [--include-tests] [--check]")
	}
	if *check && *format == "mermaid" {
		return fmt.Errorf("--check cannot be combined with --format mermaid; use text or json")
	}
	graph, err := projectgraph.Build(".", *includeTests)
	if err != nil {
		return fmt.Errorf("build dependency graph: %w", err)
	}
	switch *format {
	case "json":
		if *check {
			analysis := projectgraph.Analyze(graph)
			if err := json.NewEncoder(out).Encode(analysis); err != nil {
				return err
			}
			return graphCheckError(analysis)
		}
		return json.NewEncoder(out).Encode(graph)
	case "mermaid":
		_, err = io.WriteString(out, projectgraph.Mermaid(graph))
		return err
	default:
		p := newPalette()
		section(out, "Dependency Graph")
		fmt.Fprintf(out, "  %s %s\n", p.dim("module"), p.silver(graph.Module))
		fmt.Fprintf(out, "  %s %d  %s %d\n\n", p.dim("packages"), len(graph.Nodes), p.dim("dependencies"), len(graph.Edges))
		if len(graph.Edges) == 0 {
			fmt.Fprintf(out, "  %s\n", p.dim("No internal package dependencies found."))
		} else {
			for _, edge := range graph.Edges {
				fmt.Fprintf(out, "  %s %s %s\n", p.silver(shortPackage(graph.Module, edge.From)), p.cyan("->"), p.amber(shortPackage(graph.Module, edge.To)))
			}
		}
		if !*check {
			return nil
		}
		analysis := projectgraph.Analyze(graph)
		fmt.Fprintln(out)
		section(out, "Architecture Check")
		if analysis.OK {
			fmt.Fprintf(out, "  %s\n", p.ok("No cycles or boundary violations found."))
			return nil
		}
		for _, violation := range analysis.Violations {
			fmt.Fprintf(out, "  [%s] %s\n", p.fail(violation.Type), p.silver(violation.Message))
		}
		return graphCheckError(analysis)
	}
}

func graphCheckError(analysis projectgraph.Analysis) error {
	if !analysis.OK {
		return fmt.Errorf("dependency graph has %d architecture violation(s)", len(analysis.Violations))
	}
	return nil
}

func shortPackage(module, packagePath string) string {
	if packagePath == module {
		return filepath.Base(module)
	}
	return strings.TrimPrefix(packagePath, module+"/")
}

func runUpgrade(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	checkOnly := fs.Bool("check", false, "show the upgrade plan without applying it")
	yes := fs.Bool("yes", false, "apply without confirmation")
	jsonOutput := fs.Bool("json", false, "write machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub upgrade [--check] [--yes] [--json]")
	}
	plan, err := projectupgrade.Check(".", gokub.Version)
	if err != nil {
		return fmt.Errorf("check project upgrade: %w", err)
	}
	if *jsonOutput && (!*yes || *checkOnly) {
		return json.NewEncoder(out).Encode(plan)
	}
	if !*jsonOutput {
		renderUpgradePlan(out, plan)
	}
	if !plan.NeedsUpgrade || *checkOnly {
		return nil
	}
	if !*yes {
		choice := newPrompter(in, out, 1).choice("Apply upgrade", []string{"no", "yes"}, "no")
		if choice != "yes" {
			fmt.Fprintln(out, newPalette().dim("Upgrade cancelled. No files changed."))
			return nil
		}
	}
	result, err := projectupgrade.Apply(".", gokub.Version)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return json.NewEncoder(out).Encode(result)
	}
	success(out, "upgraded project manifest")
	fmt.Fprintf(out, "  %s %s\n", newPalette().dim("backup"), newPalette().silver(result.BackupPath))
	return nil
}

func renderUpgradePlan(out io.Writer, plan projectupgrade.Plan) {
	p := newPalette()
	section(out, "Project Upgrade")
	fmt.Fprintf(out, "  %s %d -> %d\n", p.dim("schema   "), plan.CurrentSchema, plan.TargetSchema)
	current := plan.CurrentGenerator
	if current == "" {
		current = "unversioned"
	}
	fmt.Fprintf(out, "  %s %s -> %s\n", p.dim("generator"), current, plan.TargetGenerator)
	if !plan.NeedsUpgrade {
		fmt.Fprintf(out, "\n  %s\n", p.ok("Project metadata is current."))
		return
	}
	fmt.Fprintln(out)
	for _, change := range plan.Changes {
		fmt.Fprintf(out, "  %s %s\n", p.amber("+"), p.silver(change))
	}
	fmt.Fprintf(out, "\n  %s\n", p.dim("Only .gokub.yaml changes; a backup is created before apply."))
}

func runUpdate(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	repo := fs.String("repo", repository(), "GitHub repository")
	checkOnly := fs.Bool("check", false, "check latest release without installing")
	yes := fs.Bool("yes", false, "install without confirmation")
	jsonOutput := fs.Bool("json", false, "write machine-readable output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: gokub update [--check] [--yes] [--json] [--repo owner/repo]")
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate GOKUB executable: %w", err)
	}
	plan, err := selfupdate.Check(selfupdate.Options{
		Repository: *repo, CurrentVersion: gokub.Version, Executable: executable,
	})
	if err != nil {
		return err
	}
	if *jsonOutput && (!*yes || *checkOnly) {
		return json.NewEncoder(out).Encode(plan)
	}
	if !*jsonOutput {
		renderSelfUpdatePlan(out, plan)
	}
	if !plan.Update || *checkOnly {
		return nil
	}
	if !*yes {
		selected := newPrompter(in, out, 1).choice("Install update", []string{"no", "yes"}, "no")
		if selected != "yes" {
			fmt.Fprintln(out, newPalette().dim("Update cancelled. No files changed."))
			return nil
		}
	}
	result, err := selfupdate.Apply(plan, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return json.NewEncoder(out).Encode(result)
	}
	success(out, "updated GOKUB %s -> %s", result.Previous, result.Installed)
	fmt.Fprintf(out, "  %s %s\n", newPalette().dim("binary"), newPalette().silver(result.Executable))
	return nil
}

func renderSelfUpdatePlan(out io.Writer, plan selfupdate.Plan) {
	p := newPalette()
	section(out, "CLI Update")
	fmt.Fprintf(out, "  %s %s\n  %s %s\n", p.dim("current"), p.silver(plan.Current), p.dim("latest "), p.amber(plan.Latest))
	if !plan.Update {
		fmt.Fprintf(out, "\n  %s\n", p.ok("GOKUB is up to date."))
		return
	}
	fmt.Fprintf(out, "  %s %s\n\n", p.dim("asset  "), p.silver(plan.ArchiveName))
	fmt.Fprintf(out, "  %s\n", p.dim("The release archive is checksum-verified before atomic replacement."))
}

func repository() string {
	if value := os.Getenv("GOKUB_REPOSITORY"); value != "" {
		return value
	}
	return gokub.Repository
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

// menuChoice renders an interactive choice for the looping command center. It
// returns ok=false when the input ends (EOF or a closed terminal) so the caller
// can leave the loop instead of blocking on the next keypress.
func (p *prompter) menuChoice(label string, options []string, fallback string) (string, bool) {
	step := p.nextStep()
	if p.input != nil && terminalAvailable(p.input) {
		return p.choiceInteractive(step, label, options, fallback)
	}
	return fallback, false
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
	for _, name := range []string{"go-version", "framework", "database", "messaging", "agents", "recipe"} {
		if !setFlags[name] {
			total++
		}
	}
	// A template is only asked when community templates are installed.
	if !setFlags["template"] {
		if names, err := customtemplates.Names(); err == nil && len(names) > 0 {
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
	fmt.Fprintf(out, "  %s %s %s\n", pal.dim("go          "), pal.amber(m.GoVersion), pal.dim(goversion.Description(m.GoVersion)))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("framework   "), pal.silver(m.Framework))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("database    "), pal.silver(m.Database))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("messaging   "), pal.silver(m.Messaging))
	fmt.Fprintf(out, "  %s %s\n", pal.dim("vibe coding "), pal.silver(m.Agents))
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
	commandLine(out, exe+" score", "measure architecture, security, testing, and operations")
	commandLine(out, exe+" graph", "visualize internal Go package dependencies")
	commandLine(out, exe+" upgrade", "safely migrate project metadata")
	commandLine(out, exe+" update", "checksum-verify and install a CLI release")
	commandLine(out, exe+" recipe [list|add <name>]", "manage capability bundles")
	commandLine(out, exe+" template [list|install|add|remove]", "manage local and community project templates")
	commandLine(out, exe+" plugin [list|create|install|pack|verify|run|remove]", "manage explicit local CLI plugins")
	commandLine(out, exe+" skill [list|install|remove]", "manage Codex, Claude, Copilot, and agent skills")
	commandLine(out, exe+" completion install [shell]", "enable Bash, Zsh, or Fish tab completion")
	commandLine(out, exe+" agent init [--provider <agent>]", "create AI agent guidance and MCP files")
	commandLine(out, exe+" mcp serve", "expose project tools to Codex, Claude, and MCP agents")
	commandLine(out, exe+" version [--json]", "show version, build provenance, and platform")
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
		commandLine(out, "--go-version <major.minor>", "Go version, default "+goversion.Recommended+"; "+goversion.Conservative+" is the conservative baseline")
		commandLine(out, "--template <name>", "project template, default gin-clean")
		commandLine(out, "--style <name>", "project style: monolith or microservices")
		commandLine(out, "--framework <name>", "web framework, default gin")
		commandLine(out, "--database <name>", "database provider, default postgres")
		commandLine(out, "--architecture <name>", "architecture style, default clean")
		commandLine(out, "--messaging <name>", "messaging provider, default none")
		commandLine(out, "--agents <name>", "AI assistants: all|codex|claude|copilot|gemini|none")
		commandLine(out, "--recipe <name>", "apply a recipe during creation")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Examples"))
		commandLine(out, "gokub new", "")
		commandLine(out, "gokub new payment-api --module github.com/acme/payment-api", "")
		commandLine(out, "gokub new payment-api --recipe event-driven", "")
	case "add":
		commandLine(out, "gokub add <feature> [name]", "add a capability or CRUD module")
		commandLine(out, "gokub add model", "choose a JSON file and model name interactively")
		commandLine(out, "gokub add model <name> --from <file.json>", "generate a Go model from JSON or JSON Schema")
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
		commandLine(out, "gokub add model user --from user.json", "")
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
		commandLine(out, "gokub status --json", "write project and capability state as machine-readable JSON")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Example"))
		commandLine(out, "gokub status", "")
	case "doctor":
		commandLine(out, "gokub doctor", "check project structure, manifest readability, and generated folders")
		commandLine(out, "gokub doctor --json", "write the check summary as machine-readable JSON")
	case "score":
		commandLine(out, "gokub score", "show the project health score and recommended next steps")
		commandLine(out, "gokub score --json", "write the score report as machine-readable JSON")
		commandLine(out, "gokub score --fail-under 80", "fail CI when the project score is below the threshold")
	case "graph":
		commandLine(out, "gokub graph", "show internal package dependencies")
		commandLine(out, "gokub graph --format mermaid", "write a Mermaid flowchart")
		commandLine(out, "gokub graph --format json", "write a machine-readable graph")
		commandLine(out, "--include-tests", "include imports from Go test files")
		commandLine(out, "--check", "fail on import cycles and clean-architecture boundary violations")
	case "upgrade":
		commandLine(out, "gokub upgrade", "preview and confirm a safe project migration")
		commandLine(out, "gokub upgrade --check", "show the migration plan without writing files")
		commandLine(out, "gokub upgrade --yes", "apply the migration without confirmation")
		commandLine(out, "gokub upgrade --json", "write a machine-readable migration plan")
	case "update":
		commandLine(out, "gokub update", "check, confirm, and atomically install the latest release")
		commandLine(out, "gokub update --check", "check without changing the executable")
		commandLine(out, "gokub update --yes", "install without interactive confirmation")
		commandLine(out, "gokub update --json", "write a machine-readable update plan")
		commandLine(out, "--repo <owner/repo>", "override the GitHub release repository")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Example"))
		commandLine(out, "gokub update --check --repo ongyoo/gokub", "")
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
		commandLine(out, "gokub template search [query]", "search GitHub repositories tagged gokub-template")
		commandLine(out, "gokub template search [query] --install", "choose and install a search result")
		commandLine(out, "gokub template install <owner/repo>", "install a community template from GitHub")
		commandLine(out, "--ref <tag>", "pin a Git branch or tag")
		commandLine(out, "--subdir <path>", "install one template from a repository subdirectory")
		commandLine(out, "--name <name>", "choose the local template name")
		commandLine(out, "gokub template add <path>", "install a folder using its directory name")
		commandLine(out, "gokub template add <name> <path>", "install a folder with a custom name")
		commandLine(out, "gokub template list", "list installed custom templates")
		commandLine(out, "gokub template remove <name>", "remove an installed custom template")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Template placeholders"))
		for _, placeholder := range []string{"{{project_name}}", "{{module}}", "{{template}}", "{{style}}", "{{framework}}", "{{database}}", "{{architecture}}", "{{messaging}}"} {
			commandLine(out, placeholder, "")
		}
	case "plugin":
		commandLine(out, "gokub plugin create <name>", "scaffold a versioned Go plugin")
		commandLine(out, "gokub plugin install <path>", "install a built plugin folder")
		commandLine(out, "gokub plugin pack [path]", "create a reproducible archive and SHA-256 file")
		commandLine(out, "gokub plugin verify <archive>", "verify an artifact against its checksum file")
		commandLine(out, "gokub plugin list", "list installed plugins and commands")
		commandLine(out, "gokub plugin run <name> [command]", "explicitly execute an installed plugin")
		commandLine(out, "gokub plugin remove <name>", "remove an installed plugin")
		fmt.Fprintln(out)
		fmt.Fprintln(out, p.silver("Security"))
		fmt.Fprintln(out, p.dim("  Installing does not execute a plugin. Review third-party code before running it."))
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
	case "completion":
		commandLine(out, "gokub completion install", "detect the current shell and install completion")
		commandLine(out, "gokub completion install <shell>", "install completion for bash, zsh, or fish")
		commandLine(out, "gokub completion <shell>", "print a completion script to stdout")
	case "version":
		commandLine(out, "gokub version", "show CLI version and build provenance")
		commandLine(out, "gokub version --json", "write build metadata as machine-readable JSON")
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
