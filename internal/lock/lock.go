// Package lock reads and writes .kiln/kiln.lock, the hash of every file kiln installed.
package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Path is the lock file location, relative to the repo root.
const Path = ".kiln/kiln.lock"

// Lock records the version that wrote it and a sha256 per installed file.
type Lock struct {
	Version string            `yaml:"version"`
	Files   map[string]string `yaml:"files"`
}

// Hash returns "sha256:<hex>" for b.
func Hash(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// HashFile hashes a file on disk.
func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}

// Load reads the lock under root. A missing lock returns an empty one and os.ErrNotExist.
func Load(root string) (Lock, error) {
	l := Lock{Files: map[string]string{}}
	b, err := os.ReadFile(filepath.Join(root, Path))
	if err != nil {
		return l, err
	}
	if err := yaml.Unmarshal(b, &l); err != nil {
		return l, fmt.Errorf("kiln.lock: %w", err)
	}
	if l.Files == nil {
		l.Files = map[string]string{}
	}
	return l, nil
}

// Save writes the lock under root.
func (l Lock) Save(root string) error {
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	head := []byte("# Written by kiln init and kiln update. Don't edit: kiln uses these hashes to\n# tell your edits apart from files it can safely replace.\n")
	return os.WriteFile(filepath.Join(root, Path), append(head, b...), 0o644)
}

// Decision is what update should do with one file.
type Decision string

// Update decisions.
const (
	Create   Decision = "create"   // not on disk yet
	Replace  Decision = "replace"  // untouched since kiln wrote it, and kiln has a new version
	Keep     Decision = "keep"     // already identical to the new version
	Conflict Decision = "conflict" // edited by you and kiln has a new version: write .kiln-new
	Edited   Decision = "edited"   // edited by you, kiln has nothing new
)

// Decide compares the file on disk with the lock and the new content.
// current is nil when the file doesn't exist.
func Decide(current []byte, locked string, next []byte) Decision {
	if current == nil {
		return Create
	}
	cur, nxt := Hash(current), Hash(next)
	switch {
	case cur == nxt:
		return Keep
	case locked == "":
		return Conflict // unknown origin: never overwrite
	case cur == locked:
		return Replace
	case locked == nxt:
		return Edited
	default:
		return Conflict
	}
}
