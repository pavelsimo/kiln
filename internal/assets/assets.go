// Package assets embeds the skills and templates that kiln init writes into a repo.
package assets

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed all:skills all:templates
var files embed.FS

// Skills is the skills tree, rooted so that "kiln-issue/SKILL.md" is a valid path.
func Skills() fs.FS {
	sub, err := fs.Sub(files, "skills")
	if err != nil {
		panic(err)
	}
	return sub
}

// Template returns an embedded template file, e.g. "kiln/config.yaml".
func Template(path string) []byte {
	b, err := files.ReadFile("templates/" + path)
	if err != nil {
		panic("assets: missing template " + path)
	}
	return b
}

// Templates walks a template folder and returns its files keyed by path relative to dir.
func Templates(dir string) map[string][]byte {
	out := map[string][]byte{}
	root := "templates/" + dir
	_ = fs.WalkDir(files, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := files.ReadFile(p)
		out[strings.TrimPrefix(p, root+"/")] = b
		return nil
	})
	return out
}

// SkillNames lists the top-level skill folders, sorted.
func SkillNames() []string {
	entries, err := fs.ReadDir(Skills(), ".")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// SkillFiles returns every file under skills/, keyed by path relative to skills/.
func SkillFiles() map[string][]byte {
	out := map[string][]byte{}
	s := Skills()
	_ = fs.WalkDir(s, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := fs.ReadFile(s, p)
		out[p] = b
		return nil
	})
	return out
}
