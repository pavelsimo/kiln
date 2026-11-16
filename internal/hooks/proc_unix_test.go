//go:build unix

package hooks

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// "sleep 30; true" makes the shell fork for sleep, so killing only the shell would leave it running.
func TestTimeoutKillsWhatTheHookStarted(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	h := Hook{On: "move", Run: `sh -c 'echo $$ > "` + pidFile + `"; exec sleep 30'; true`, Timeout: 300 * time.Millisecond}
	res := Run(context.Background(), []Hook{h}, nil, nil, t.TempDir(), io.Discard)
	if res[0].OK || res[0].Error != "timed out after 300ms" {
		t.Fatalf("result = %+v", res[0])
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	for range 50 { // the kernel reaps it a moment after the kill
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Errorf("the hook's child (pid %d) outlived the timeout", pid)
}
