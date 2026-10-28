// Package journal reads kiln task journals (.kiln/memory/tasks/<n>-<slug>.md).
package journal

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Dir is where task journals live, relative to the repo root.
const Dir = ".kiln/memory/tasks"

// Meta is the journal's frontmatter.
type Meta struct {
	Issue    int    `yaml:"issue" json:"issue"`
	Title    string `yaml:"title" json:"title"`
	Type     string `yaml:"type" json:"type"`
	Size     string `yaml:"size" json:"size"`
	Playbook string `yaml:"playbook" json:"playbook"`
	Branch   string `yaml:"branch" json:"branch"`
	PR       int    `yaml:"pr" json:"pr,omitempty"`
}

// Row is one journal table row.
type Row struct {
	TS     string `json:"ts"`
	Skill  string `json:"skill"`
	Action string `json:"action"`
	Ref    string `json:"ref"`
	Result string `json:"result"`
}

// Time parses the row timestamp ("2006-01-02 15:04" or a date).
func (r Row) Time() (time.Time, bool) {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04", time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, strings.TrimSpace(r.TS)); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// Journal is a parsed task journal.
type Journal struct {
	Meta       Meta     `json:"meta"`
	Rows       []Row    `json:"rows"`
	RetroNotes []string `json:"retro_notes"`
	Skips      []string `json:"skips"`
}

// Parse reads a journal file's contents.
func Parse(src string) (Journal, error) {
	var j Journal
	body := src
	if rest, ok := strings.CutPrefix(strings.TrimPrefix(src, "\ufeff"), "---\n"); ok {
		front, after, found := strings.Cut(rest, "\n---")
		if !found {
			return j, fmt.Errorf("journal: unterminated frontmatter")
		}
		if err := yaml.Unmarshal([]byte(front), &j.Meta); err != nil {
			return j, fmt.Errorf("journal: frontmatter: %w", err)
		}
		body = after
	}
	section := ""
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if h, ok := strings.CutPrefix(line, "## "); ok {
			section = strings.ToLower(strings.TrimSpace(h))
			continue
		}
		switch section {
		case "journal":
			if r, ok := parseRow(line); ok {
				j.Rows = append(j.Rows, r)
			}
		case "retro notes":
			if b, ok := bullet(line); ok {
				j.RetroNotes = append(j.RetroNotes, b)
			}
		case "skipped":
			if b, ok := bullet(line); ok {
				j.Skips = append(j.Skips, b)
			}
		}
	}
	return j, sc.Err()
}

func bullet(line string) (string, bool) {
	for _, p := range []string{"- ", "* "} {
		if b, ok := strings.CutPrefix(line, p); ok && strings.TrimSpace(b) != "" {
			return strings.TrimSpace(b), true
		}
	}
	return "", false
}

func parseRow(line string) (Row, bool) {
	if !strings.HasPrefix(line, "|") {
		return Row{}, false
	}
	cells := strings.Split(strings.Trim(line, "|"), "|")
	if len(cells) < 5 {
		return Row{}, false
	}
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	if strings.EqualFold(cells[0], "ts") || strings.Trim(cells[0], "-: ") == "" {
		return Row{}, false
	}
	return Row{TS: cells[0], Skill: cells[1], Action: cells[2], Ref: strings.Trim(cells[3], "`"), Result: cells[4]}, true
}

// Latest returns the last row written by skill.
func (j Journal) Latest(skill string) (Row, bool) {
	for i := len(j.Rows) - 1; i >= 0; i-- {
		if j.Rows[i].Skill == skill {
			return j.Rows[i], true
		}
	}
	return Row{}, false
}

// LastActivity returns the newest row timestamp.
func (j Journal) LastActivity() (time.Time, bool) {
	var best time.Time
	ok := false
	for _, r := range j.Rows {
		if t, good := r.Time(); good && t.After(best) {
			best, ok = t, true
		}
	}
	return best, ok
}

// Normalize canonicalizes a verdict string ("verified" → "VERIFIED").
func Normalize(result string) string {
	r := strings.ToUpper(strings.TrimSpace(result))
	r = strings.ReplaceAll(r, "_", " ")
	return strings.Join(strings.Fields(r), " ")
}
