package journal

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pavelsimo/kiln/internal/gitx"
)

// matches reports whether a journal file name belongs to issue n ("42-dark-mode.md").
func matches(name string, n int) bool {
	base := path.Base(name)
	return strings.HasSuffix(base, ".md") && (strings.HasPrefix(base, strconv.Itoa(n)+"-") || base == strconv.Itoa(n)+".md")
}

// AtRev loads issue n's journal from a commit. It returns the path it read.
func AtRev(ctx context.Context, g *gitx.Git, rev string, n int) (Journal, string, error) {
	names, err := g.ListTree(ctx, rev, Dir)
	if err != nil {
		return Journal{}, "", fmt.Errorf("no task journals at %s: %w", gitx.Short(rev), err)
	}
	for _, name := range names {
		if matches(name, n) {
			src, err := g.Show(ctx, rev, name)
			if err != nil {
				return Journal{}, "", err
			}
			j, err := Parse(src)
			return j, name, err
		}
	}
	return Journal{}, "", fmt.Errorf("no journal for #%d in %s at %s", n, Dir, gitx.Short(rev))
}

// InDir loads every journal from a working tree (root/.kiln/memory/tasks).
func InDir(root string) (map[string]Journal, error) {
	dir := filepath.Join(root, Dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Journal{}, nil
		}
		return nil, err
	}
	out := map[string]Journal{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if j, err := Parse(string(b)); err == nil {
			out[e.Name()] = j
		}
	}
	return out, nil
}
