package install

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pavelsimo/kiln/internal/assets"
	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/lock"
)

// NewSuffix is appended to files you edited when kiln has a newer version of them.
const NewSuffix = ".kiln-new"

// Update upgrades installed skills to the ones embedded in this binary.
// Files whose hash still matches kiln.lock are replaced. Files you edited are kept,
// and the new version is written next to them with a .kiln-new suffix.
func Update(_ context.Context, opts Options) (Report, error) {
	rep := Report{Root: opts.Root, DryRun: opts.DryRun}
	w := writer{root: opts.Root, dryRun: opts.DryRun, rep: &rep}
	lk, err := lock.Load(opts.Root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return rep, fmt.Errorf("%s not found (run kiln init first)", lock.Path)
		}
		return rep, err
	}
	cfg, err := config.Load(opts.Root)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return rep, err
	}
	agents := cfg.Agents
	if len(opts.Agents) > 0 {
		agents = opts.Agents
	}
	rep.Agents = agents

	embedded := assets.SkillFiles()
	next := lock.Lock{Version: opts.Version, Files: map[string]string{}}
	for _, name := range sortedKeys(embedded) {
		data := embedded[name]
		rel := SkillsDir + "/" + name
		key := "skills/" + name
		cur, err := os.ReadFile(w.abs(rel))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return rep, err
		}
		if errors.Is(err, fs.ErrNotExist) {
			cur = nil
		}
		next.Files[key] = lock.Hash(data)
		switch lock.Decide(cur, lk.Files[key], data) {
		case lock.Create:
			err = w.write("create", rel, data)
		case lock.Replace:
			err = w.write("replace", rel, data)
		case lock.Keep:
			rep.add("keep", rel, "")
			_ = w.remove(rel + NewSuffix)
		case lock.Edited:
			rep.add("keep", rel, "your edits, nothing new from kiln")
			next.Files[key] = lk.Files[key]
		case lock.Conflict:
			err = w.write("conflict", rel+NewSuffix, data)
		}
		if err != nil {
			return rep, err
		}
	}
	rep.Skills = len(assets.SkillNames())

	// skills kiln no longer ships
	for key, h := range lk.Files {
		if _, ok := next.Files[key]; ok || !strings.HasPrefix(key, "skills/") {
			continue
		}
		rel := SkillsDir + "/" + strings.TrimPrefix(key, "skills/")
		cur, err := lock.HashFile(w.abs(rel))
		switch {
		case err != nil:
			continue
		case cur == h:
			rep.add("remove", rel, "kiln no longer ships it")
			if err := w.remove(rel); err != nil {
				return rep, err
			}
		default:
			rep.add("keep", rel, "kiln no longer ships it, but you edited it")
		}
	}
	if err := pruneEmptySkillDirs(w); err != nil {
		return rep, err
	}

	if err := linkAll(w, agents, assets.SkillNames(), false); err != nil {
		return rep, err
	}
	if err := unlinkStale(w, agents); err != nil {
		return rep, err
	}
	if err := pointer(w, agents); err != nil {
		return rep, err
	}
	if !opts.DryRun {
		if err := next.Save(opts.Root); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

func (w writer) remove(rel string) error {
	if w.dryRun {
		return nil
	}
	err := os.Remove(w.abs(rel))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func pruneEmptySkillDirs(w writer) error {
	if w.dryRun {
		return nil
	}
	root := w.abs(SkillsDir)
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != root {
			dirs = append(dirs, p)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		if entries, err := os.ReadDir(dirs[i]); err == nil && len(entries) == 0 {
			if err := os.Remove(dirs[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// unlinkStale removes agent symlinks that point at kiln skills that no longer exist.
func unlinkStale(w writer, agents []string) error {
	for _, a := range agents {
		dir := AgentDirs[a]
		entries, err := os.ReadDir(w.abs(dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			p := w.abs(dir + "/" + e.Name())
			fi, err := os.Lstat(p)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				continue
			}
			target, _ := os.Readlink(p)
			if filepath.ToSlash(target) != linkTarget(dir, e.Name()) {
				continue
			}
			if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
				w.rep.add("unlink", dir+"/"+e.Name(), "skill removed")
				if !w.dryRun {
					if err := os.Remove(p); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
