// Package ui prints kiln's human output: data to stdout, progress and errors to stderr.
package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// UI writes human or JSON output.
type UI struct {
	Out   io.Writer
	Err   io.Writer
	Color bool
}

// New returns a UI on stdout/stderr. Color is on only for a terminal without NO_COLOR.
func New(noColor bool) *UI {
	color := !noColor && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && isTerminal(os.Stdout)
	return &UI{Out: os.Stdout, Err: os.Stderr, Color: color}
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	ember  = "\033[38;5;208m"
)

func (u *UI) paint(code, s string) string {
	if !u.Color {
		return s
	}
	return code + s + reset
}

// Bold styles s in bold.
func (u *UI) Bold(s string) string { return u.paint(bold, s) }

// Dim styles s faint.
func (u *UI) Dim(s string) string { return u.paint(dim, s) }

// Ember styles s in kiln's orange.
func (u *UI) Ember(s string) string { return u.paint(ember, s) }

// Green styles s green.
func (u *UI) Green(s string) string { return u.paint(green, s) }

// Yellow styles s yellow.
func (u *UI) Yellow(s string) string { return u.paint(yellow, s) }

// Red styles s red.
func (u *UI) Red(s string) string { return u.paint(red, s) }

// Println writes a line of data to stdout.
func (u *UI) Println(a ...any) { fmt.Fprintln(u.Out, a...) }

// Printf writes data to stdout.
func (u *UI) Printf(format string, a ...any) { fmt.Fprintf(u.Out, format, a...) }

// OK prints a ✓ line to stdout.
func (u *UI) OK(format string, a ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.Green("✓"), fmt.Sprintf(format, a...))
}

// Warn prints a ! line to stdout.
func (u *UI) Warn(format string, a ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.Yellow("!"), fmt.Sprintf(format, a...))
}

// Fail prints a ✗ line to stdout.
func (u *UI) Fail(format string, a ...any) {
	fmt.Fprintf(u.Out, "  %s %s\n", u.Red("✗"), fmt.Sprintf(format, a...))
}

// Note prints a dimmed, indented line to stdout.
func (u *UI) Note(format string, a ...any) {
	fmt.Fprintf(u.Out, "    %s\n", u.Dim(fmt.Sprintf(format, a...)))
}

// Progress prints to stderr.
func (u *UI) Progress(format string, a ...any) {
	fmt.Fprintf(u.Err, format+"\n", a...)
}

// JSON writes v as indented JSON to stdout.
func (u *UI) JSON(v any) error {
	enc := json.NewEncoder(u.Out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Pad right-pads s to width runes.
func Pad(s string, width int) string {
	n := len([]rune(s))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}
