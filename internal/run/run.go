// Package run executes external commands (git, gh) and lets tests swap in fakes.
package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Runner runs a command in dir and returns its stdout.
type Runner interface {
	Run(ctx context.Context, dir string, stdin []byte, name string, args ...string) ([]byte, error)
}

// Error is returned when a command exits non-zero.
type Error struct {
	Cmd      string
	ExitCode int
	Stderr   string
}

func (e *Error) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return fmt.Sprintf("%s: exit %d", e.Cmd, e.ExitCode)
	}
	return fmt.Sprintf("%s: %s", e.Cmd, msg)
}

// Exec runs real processes. When Trace is set, every command is echoed to it.
type Exec struct {
	Trace io.Writer
}

// Run implements Runner.
func (x Exec) Run(ctx context.Context, dir string, stdin []byte, name string, args ...string) ([]byte, error) {
	line := name + " " + strings.Join(args, " ")
	if x.Trace != nil {
		fmt.Fprintf(x.Trace, "$ %s\n", line)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return out.Bytes(), &Error{Cmd: line, ExitCode: ee.ExitCode(), Stderr: errb.String()}
	}
	return out.Bytes(), nil
}

// Fake answers commands from a table keyed by "name arg1 arg2 ...".
// A key ending in "*" matches any command with that prefix.
type Fake struct {
	Responses map[string]Response
	Calls     []string
}

// Response is a canned result for Fake.
type Response struct {
	Out  string
	Err  error
	Exit int
}

// Run implements Runner.
func (f *Fake) Run(_ context.Context, _ string, _ []byte, name string, args ...string) ([]byte, error) {
	line := strings.TrimSpace(name + " " + strings.Join(args, " "))
	f.Calls = append(f.Calls, line)
	r, ok := f.Responses[line]
	if !ok {
		best := ""
		for k, v := range f.Responses {
			p, isPrefix := strings.CutSuffix(k, "*")
			if isPrefix && strings.HasPrefix(line, p) && len(p) > len(best) {
				best, r, ok = p, v, true
			}
		}
	}
	if !ok {
		return nil, &Error{Cmd: line, ExitCode: 127, Stderr: "fake: no response for " + line}
	}
	if r.Err != nil {
		return []byte(r.Out), r.Err
	}
	if r.Exit != 0 {
		return []byte(r.Out), &Error{Cmd: line, ExitCode: r.Exit}
	}
	return []byte(r.Out), nil
}
