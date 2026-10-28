package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecide(t *testing.T) {
	v1, v2, mine := []byte("v1"), []byte("v2"), []byte("mine")
	tests := []struct {
		name    string
		current []byte
		locked  string
		next    []byte
		want    Decision
	}{
		{"missing", nil, Hash(v1), v2, Create},
		{"untouched, new version", v1, Hash(v1), v2, Replace},
		{"already current", v2, Hash(v1), v2, Keep},
		{"edited, nothing new", mine, Hash(v2), v2, Edited},
		{"edited, new version", mine, Hash(v1), v2, Conflict},
		{"unknown origin", mine, "", v2, Conflict},
	}
	for _, tt := range tests {
		if got := Decide(tt.current, tt.locked, tt.next); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".kiln"), 0o755); err != nil {
		t.Fatal(err)
	}
	l := Lock{Version: "0.1.0", Files: map[string]string{"skills/kiln/SKILL.md": Hash([]byte("x"))}}
	if err := l.Save(root); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil || got.Version != "0.1.0" || got.Files["skills/kiln/SKILL.md"] != l.Files["skills/kiln/SKILL.md"] {
		t.Fatalf("got %+v, %v", got, err)
	}
}
