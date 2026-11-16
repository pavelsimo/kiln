//go:build !unix

package hooks

import "os/exec"

// ownGroup does nothing here: only the hook's shell is killed on a timeout.
func ownGroup(*exec.Cmd) {}
