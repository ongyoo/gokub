package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var supported = map[string]bool{"bash": true, "zsh": true, "fish": true}

// Detect returns a supported shell from $SHELL.
func Detect() (string, error) {
	shell := filepath.Base(os.Getenv("SHELL"))
	if !supported[shell] {
		return "", fmt.Errorf("cannot detect a supported shell from $SHELL=%q; choose bash, zsh, or fish", os.Getenv("SHELL"))
	}
	return shell, nil
}

// Script renders a standalone completion definition for shell.
func Script(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashScript, nil
	case "zsh":
		return zshScript, nil
	case "fish":
		return fishScript, nil
	default:
		return "", fmt.Errorf("unsupported shell %q; choose bash, zsh, or fish", shell)
	}
}

// Install writes completion into the user's config and returns the installed path.
func Install(shell string) (string, error) {
	if shell == "" {
		var err error
		shell, err = Detect()
		if err != nil {
			return "", err
		}
	}
	script, err := Script(shell)
	if err != nil {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	var path string
	switch shell {
	case "fish":
		path = filepath.Join(home, ".config", "fish", "completions", "gokub.fish")
	case "zsh":
		path = filepath.Join(home, ".gokub", "completions", "_gokub")
	case "bash":
		path = filepath.Join(home, ".gokub", "completions", "gokub.bash")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create completion directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		return "", fmt.Errorf("write completion: %w", err)
	}
	if shell == "fish" {
		return path, nil
	}
	rc := filepath.Join(home, "."+shell+"rc")
	line := "source " + shellQuote(path)
	if shell == "zsh" {
		line = "fpath=(" + shellQuote(filepath.Dir(path)) + " $fpath)\nautoload -Uz compinit && compinit"
	}
	if err := appendOnce(rc, "# GOKUB completion\n"+line); err != nil {
		return "", err
	}
	return path, nil
}

func appendOnce(path, block string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if strings.Contains(string(content), "# GOKUB completion") {
		return nil
	}
	prefix := ""
	if len(content) > 0 && content[len(content)-1] != '\n' {
		prefix = "\n"
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	if _, err := fmt.Fprintf(file, "%s\n%s\n", prefix, block); err != nil {
		return fmt.Errorf("update %s: %w", path, err)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

const commands = "new add remove enable disable switch status doctor score graph upgrade update recipe agent template plugin skill mcp completion uninstall version help"
const features = "auth crud postgres mongodb redis kafka rabbitmq nats grpc cron email websocket otel docker github-actions outbox model"
const capabilities = "authentication cache database messaging observability infrastructure"

var bashScript = `# bash completion for GOKUB
_gokub() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  case "$prev" in
    gokub) COMPREPLY=( $(compgen -W "` + commands + `" -- "$cur") ) ;;
    add|remove) COMPREPLY=( $(compgen -W "` + features + `" -- "$cur") ) ;;
    enable|disable|switch) COMPREPLY=( $(compgen -W "` + capabilities + `" -- "$cur") ) ;;
    completion) COMPREPLY=( $(compgen -W "bash zsh fish install" -- "$cur") ) ;;
    recipe) COMPREPLY=( $(compgen -W "list add" -- "$cur") ) ;;
    template) COMPREPLY=( $(compgen -W "list search install add remove" -- "$cur") ) ;;
    plugin) COMPREPLY=( $(compgen -W "list create install pack verify run remove" -- "$cur") ) ;;
    skill|skills) COMPREPLY=( $(compgen -W "list install remove" -- "$cur") ) ;;
    *) [[ "$cur" == -* ]] && COMPREPLY=( $(compgen -W "--help --version" -- "$cur") ) ;;
  esac
}
complete -F _gokub gokub
`

var zshScript = `#compdef gokub
_gokub() {
  local -a commands
  commands=(
    'new:create a Go project' 'add:add a feature' 'remove:remove a feature'
    'enable:enable a capability' 'disable:disable a capability' 'switch:switch provider'
    'status:show project state' 'doctor:check project health' 'score:score project health'
    'graph:show dependencies' 'upgrade:upgrade project metadata' 'update:update GOKUB'
    'recipe:manage recipes' 'agent:configure AI agents' 'template:manage templates'
    'plugin:manage plugins' 'skill:manage agent skills' 'mcp:start MCP server'
    'completion:install shell completion' 'uninstall:remove GOKUB' 'version:show version' 'help:show help'
  )
  if (( CURRENT == 2 )); then _describe 'command' commands; return; fi
  case $words[2] in
    add|remove) compadd ` + features + ` ;;
    enable|disable|switch) compadd ` + capabilities + ` ;;
    completion) compadd bash zsh fish install ;;
    recipe) compadd list add ;;
    template) compadd list search install add remove ;;
    plugin) compadd list create install pack verify run remove ;;
    skill|skills) compadd list install remove ;;
  esac
}
_gokub "$@"
`

var fishScript = `# fish completion for GOKUB
complete -c gokub -f
complete -c gokub -n '__fish_use_subcommand' -a '` + commands + `'
complete -c gokub -n '__fish_seen_subcommand_from add remove' -a '` + features + `'
complete -c gokub -n '__fish_seen_subcommand_from enable disable switch' -a '` + capabilities + `'
complete -c gokub -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish install'
complete -c gokub -n '__fish_seen_subcommand_from recipe' -a 'list add'
complete -c gokub -n '__fish_seen_subcommand_from template' -a 'list search install add remove'
complete -c gokub -n '__fish_seen_subcommand_from plugin' -a 'list create install pack verify run remove'
complete -c gokub -n '__fish_seen_subcommand_from skill skills' -a 'list install remove'
`
