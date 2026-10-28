package config

import (
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	c := Default()
	if errs := c.Validate(); len(errs) > 0 {
		t.Fatalf("default config is invalid: %v", errs)
	}
	if c.Autonomy != "gated" || c.Worktrees.MaxInFlight != 3 || c.Rigor["L"].ForceGates[0] != "plan" {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestParseLayersOnDefaults(t *testing.T) {
	c, err := Parse([]byte("autonomy: autopilot\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Autonomy != "autopilot" || c.Release.Tag != "v{version}" {
		t.Fatalf("got %+v", c)
	}
	if _, err := Parse([]byte("autonmy: gated\n")); err == nil {
		t.Fatal("unknown keys should be rejected")
	}
}

func TestValidate(t *testing.T) {
	c, _ := Parse([]byte("autonomy: yolo\ngates: [plan, deploy]\nagents: [claude, vim]\nworktrees: { max_in_flight: 0 }\n"))
	msg := ""
	for _, e := range c.Validate() {
		msg += e.Error() + "\n"
	}
	for _, want := range []string{"autonomy", "deploy", "vim", "max_in_flight"} {
		if !strings.Contains(msg, want) {
			t.Errorf("want an error mentioning %q, got:\n%s", want, msg)
		}
	}
}

func TestGateActive(t *testing.T) {
	c := Default()
	if !c.GateActive("plan", "S") || !c.GateActive("merge", "M") {
		t.Fatal("gated stops at both gates")
	}
	c.Autonomy = "autopilot"
	if c.GateActive("merge", "M") || c.GateActive("plan", "M") {
		t.Fatal("autopilot skips gates for M")
	}
	if !c.GateActive("plan", "L") {
		t.Fatal("L forces the plan gate")
	}
	c.Autonomy = "manual"
	if !c.GateActive("merge", "S") {
		t.Fatal("manual stops everywhere")
	}
}

func TestWorktreeDir(t *testing.T) {
	if got := Default().WorktreeDir("/src/app"); got != "/src/app.kiln" {
		t.Fatalf("got %s", got)
	}
}
