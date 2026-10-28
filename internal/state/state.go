// Package state is the kiln:* label state machine that kiln move enforces.
package state

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// States in lifecycle order.
const (
	None        = ""
	NeedsTriage = "needs-triage"
	Ready       = "ready"
	Planned     = "planned"
	InProgress  = "in-progress"
	Review      = "review"
	Blocked     = "blocked"
)

// All lists every state kiln can move an issue to.
var All = []string{NeedsTriage, Ready, Planned, InProgress, Review, Blocked}

// Label returns the GitHub label for a state.
func Label(s string) string { return "kiln:" + s }

var forward = map[string][]string{
	None:        {NeedsTriage, Ready},
	NeedsTriage: {Ready},
	Ready:       {Planned, InProgress},
	Planned:     {InProgress, Ready},
	InProgress:  {Review},
	Review:      {InProgress},
}

// Valid reports whether s names a state.
func Valid(s string) bool { return slices.Contains(All, s) }

// Current returns the kiln state among labels. More than one kiln:* label is an error.
func Current(labels []string) (string, error) {
	var found []string
	for _, l := range labels {
		if s, ok := strings.CutPrefix(l, "kiln:"); ok && Valid(s) {
			found = append(found, s)
		}
	}
	switch len(found) {
	case 0:
		return None, nil
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("issue has %d kiln:* labels (%s); fix them by hand first", len(found), strings.Join(found, ", "))
	}
}

// Check returns nil if from → to is legal. blockedFrom is the state recorded when the
// issue was blocked ("" if unknown).
func Check(from, to, blockedFrom string) error {
	if !Valid(to) {
		return fmt.Errorf("unknown state %q (want one of %s)", to, strings.Join(All, ", "))
	}
	if from == to {
		return fmt.Errorf("issue is already %s", Label(to))
	}
	if to == Blocked {
		if from == None {
			return fmt.Errorf("issue has no kiln:* label yet, so it can't be blocked")
		}
		return nil
	}
	if from == Blocked {
		if blockedFrom == "" || blockedFrom == to {
			return nil
		}
		return fmt.Errorf("issue was blocked from %s, so it can only go back there", Label(blockedFrom))
	}
	if slices.Contains(forward[from], to) {
		return nil
	}
	return fmt.Errorf("%s → %s isn't a legal move (from %s you can go to %s)", show(from), Label(to), show(from), strings.Join(Next(from), ", "))
}

// Next lists the states reachable from s.
func Next(s string) []string {
	var out []string
	for _, n := range forward[s] {
		out = append(out, Label(n))
	}
	if s != None && s != Blocked {
		out = append(out, Label(Blocked))
	}
	return out
}

func show(s string) string {
	if s == None {
		return "(no kiln label)"
	}
	return Label(s)
}

// BlockedMarker is embedded in the comment kiln move posts when blocking an issue.
func BlockedMarker(from string) string { return "<!-- kiln:blocked-from=" + from + " -->" }

var markerRe = regexp.MustCompile(`<!-- kiln:blocked-from=([a-z-]+) -->`)

// BlockedFrom finds the most recent blocked-from marker in comment bodies (oldest first).
func BlockedFrom(bodies []string) string {
	for i := len(bodies) - 1; i >= 0; i-- {
		if m := markerRe.FindStringSubmatch(bodies[i]); m != nil {
			return m[1]
		}
	}
	return ""
}

// Stage maps a state to its lifecycle stage name.
func Stage(s string) string {
	switch s {
	case NeedsTriage, Ready:
		return "gather"
	case Planned:
		return "shape"
	case InProgress:
		return "fire"
	case Review:
		return "glaze"
	case Blocked:
		return "blocked"
	}
	return ""
}
