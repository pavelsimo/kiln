package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/assets"
)

var (
	codeSpan = regexp.MustCompile("`([^`\n]+)`")
	// kiln as a command word: not .kiln/, kiln-verify, kiln:ready or kiln/42-x
	kilnCall = regexp.MustCompile(`(?:^|[^\w./:-])kiln[ \t]+(.+)$`)
	cmdWord  = regexp.MustCompile(`^[a-z][a-z-]*$`)
	longFlag = regexp.MustCompile(`^--([a-z][a-z-]*)`)
	shellSep = regexp.MustCompile(`&&|\|\||[;|#]`)
)

// skillCode returns the code in a skill file: every fenced line and every inline code span.
// Prose is never returned, so "kiln wrote the file" can't be mistaken for a command.
func skillCode(md string) []string {
	var out []string
	fenced := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			out = append(out, line)
			continue
		}
		for _, m := range codeSpan.FindAllStringSubmatch(line, -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// kilnCalls splits code into the kiln invocations it holds, as the words after "kiln".
func kilnCalls(code string) [][]string {
	var calls [][]string
	for _, part := range shellSep.Split(code, -1) {
		if m := kilnCall.FindStringSubmatch(part); m != nil {
			calls = append(calls, strings.Fields(m[1]))
		}
	}
	return calls
}

// parseCall separates a call into its command path and the long flags it passes.
func parseCall(words []string) (path, flags []string) {
	inPath := true
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch {
		case w == "-C" || w == "--dir":
			i++ // skip the directory
		case longFlag.MatchString(w):
			flags = append(flags, longFlag.FindStringSubmatch(w)[1])
			inPath = len(path) == 0 // global flags may come before the command
		case inPath && cmdWord.MatchString(w):
			path = append(path, w)
		default:
			inPath = false
		}
	}
	return path, flags
}

func hasFlag(c *cobra.Command, name string) bool {
	if name == "help" || (name == "version" && c == rootCmd) {
		return true
	}
	return c.Flags().Lookup(name) != nil || c.PersistentFlags().Lookup(name) != nil || c.InheritedFlags().Lookup(name) != nil
}

// Every kiln command and flag a skill tells the agent to run must exist in the binary.
func TestSkillsMatchCLI(t *testing.T) {
	calls := 0
	for file, body := range assets.SkillFiles() {
		if !strings.HasSuffix(file, ".md") {
			continue
		}
		for _, code := range skillCode(string(body)) {
			for _, words := range kilnCalls(code) {
				path, flags := parseCall(words)
				c := rootCmd
				if len(path) > 0 {
					found, _, _ := rootCmd.Find(path)
					if found == nil || found == rootCmd {
						t.Errorf("%s: `kiln %s`: no such command %q", file, strings.Join(words, " "), path[0])
						continue
					}
					c = found
				} else if len(flags) == 0 {
					continue // a placeholder such as kiln <cmd>
				}
				calls++
				for _, f := range flags {
					if !hasFlag(c, f) {
						t.Errorf("%s: `kiln %s`: %q has no flag --%s", file, strings.Join(words, " "), c.CommandPath(), f)
					}
				}
			}
		}
	}
	if calls == 0 {
		t.Fatal("found no kiln calls in the skills; the extraction is broken")
	}
	t.Logf("checked %d kiln calls", calls)
}
