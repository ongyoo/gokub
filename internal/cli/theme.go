package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// spinner is a lightweight terminal loading indicator used while GOKUB works.
type spinner struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

// startSpinner animates a labeled loading indicator on a terminal. When out is
// not an interactive terminal it does nothing, so scripted and CI output stays
// clean.
func startSpinner(out io.Writer, file *os.File, label string) *spinner {
	s := &spinner{stop: make(chan struct{}), done: make(chan struct{})}
	if file == nil || !terminalAvailable(file) {
		close(s.done)
		return s
	}
	go func() {
		defer close(s.done)
		p := newPalette()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		fmt.Fprint(out, "\x1b[?25l") // hide cursor
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				fmt.Fprint(out, "\r\x1b[K\x1b[?25h") // clear line, restore cursor
				return
			default:
				fmt.Fprintf(out, "\r  %s %s %s", p.cyan(frames[i%len(frames)]), p.amber("GoKub"), p.dim(label))
				time.Sleep(90 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop ends the animation and clears the line. Safe to call once.
func (s *spinner) Stop() {
	s.once.Do(func() {
		select {
		case <-s.done:
			return
		default:
			close(s.stop)
			<-s.done
		}
	})
}

type palette struct {
	enabled bool
}

func newPalette() palette {
	return palette{enabled: os.Getenv("NO_COLOR") == ""}
}

func (p palette) cyan(value string) string   { return p.wrap("36;1", value) }
func (p palette) amber(value string) string  { return p.wrap("33;1", value) }
func (p palette) silver(value string) string { return p.wrap("37;1", value) }
func (p palette) dim(value string) string    { return p.wrap("2", value) }
func (p palette) ok(value string) string     { return p.wrap("36;1", value) }
func (p palette) fail(value string) string   { return p.wrap("31;1", value) }
func (p palette) selected(value string) string {
	return p.wrap("30;46;1", value)
}

func (p palette) wrap(code, value string) string {
	if !p.enabled {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func banner(out io.Writer) {
	p := newPalette()
	fmt.Fprintln(out, p.silver("Build production-ready Go services in minutes, not days."))
	fmt.Fprintln(out)
}

func startupLogo(out io.Writer, version string) {
	logo(out, version, "Go Project Kit")
	fmt.Fprintln(out)
}

func logo(out io.Writer, version, detail string) {
	p := newPalette()
	fmt.Fprintln(out, p.dim("   ╭────────────────────────────────────────────────────────╮"))
	fmt.Fprintln(out, p.silver("             __")+p.amber("/\\")+p.silver("__                         __")+p.amber("/\\")+p.silver("__"))
	fmt.Fprintln(out, p.silver("          __/  \\__")+p.amber("\\")+p.silver("        ")+p.cyan("◇")+p.silver("        ")+p.amber("/")+p.silver("__/  \\__"))
	fmt.Fprintln(out, p.silver("         /__  __  \\")+p.amber("╲")+p.cyan("▣")+p.amber("╱")+p.silver("  ")+p.cyan("GOKUB")+p.silver("  ")+p.amber("╲")+p.cyan("▣")+p.amber("╱")+p.silver("/  __  __\\"))
	fmt.Fprintln(out, p.silver("            \\/  \\_")+p.amber("╱")+p.cyan("●")+p.amber("╲")+p.silver("________")+p.amber("╱")+p.cyan("●")+p.amber("╲")+p.silver("_/  \\/"))
	fmt.Fprintln(out, p.cyan("      ██████╗  ██████╗ ██╗  ██╗██╗   ██╗██████╗"))
	fmt.Fprintln(out, p.silver("     ██╔════╝ ██╔═══██╗██║ ██╔╝██║   ██║██╔══██╗"))
	fmt.Fprintln(out, p.silver("     ██║  ███╗██║   ██║█████╔╝ ██║   ██║██████╔╝"))
	fmt.Fprintln(out, p.amber("     ██║   ██║██║   ██║██╔═██╗ ██║   ██║██╔══██╗"))
	fmt.Fprintln(out, p.silver("     ╚██████╔╝╚██████╔╝██║  ██╗╚██████╔╝██████╔╝"))
	fmt.Fprintln(out, p.silver("      ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═════╝"))
	fmt.Fprintln(out, p.dim("   ╰────────────────────────────────────────────────────────╯"))
	if version != "" {
		line := "      " + p.amber("GOKUB") + "  " + p.cyan("version ") + p.silver(version)
		if detail != "" {
			line += p.dim("  " + detail)
		}
		fmt.Fprintln(out, line)
		fmt.Fprintln(out, "      "+p.dim("Powered by ")+p.silver("Roomkub  ")+p.cyan("https://www.roomkub.com"))
	}
}

func section(out io.Writer, title string) {
	p := newPalette()
	fmt.Fprintln(out, p.cyan(title))
}

func commandLine(out io.Writer, command, description string) {
	p := newPalette()
	if description == "" {
		fmt.Fprintf(out, "  %s\n", p.amber(command))
		return
	}
	fmt.Fprintf(out, "  %s  %s\n", p.amber(command), p.dim(description))
}

func success(out io.Writer, format string, values ...any) {
	p := newPalette()
	fmt.Fprintf(out, p.ok("done")+" "+format+"\n", values...)
}
