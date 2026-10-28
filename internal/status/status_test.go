package status

import (
	"testing"

	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/ghx"
)

func TestClassify(t *testing.T) {
	cfg := config.Default()
	green := &PRInfo{Number: 58, Checks: ghx.CheckSummary{Passed: 3, Total: 3}, Mergeable: "MERGEABLE"}
	running := &PRInfo{Number: 57, Checks: ghx.CheckSummary{Passed: 4, Total: 6, Pending: 2}, Mergeable: "MERGEABLE"}
	tests := []struct {
		autonomy, state, size string
		pr                    *PRInfo
		waiting, stage        string
	}{
		{"gated", "planned", "L", nil, "plan", "shape"},
		{"autopilot", "planned", "M", nil, "", "shape"},
		{"autopilot", "planned", "L", nil, "plan", "shape"},
		{"gated", "review", "S", green, "merge", "ship"},
		{"autopilot", "review", "S", green, "", "ship"},
		{"gated", "review", "M", running, "", "glaze"},
		{"gated", "in-progress", "M", nil, "", "fire"},
	}
	for _, tt := range tests {
		cfg.Autonomy = tt.autonomy
		task := Task{Issue: 1, State: tt.state, Size: tt.size, PR: tt.pr, Stage: map[string]string{"planned": "shape", "review": "glaze", "in-progress": "fire"}[tt.state]}
		classify(&task, cfg)
		if task.Waiting != tt.waiting || task.Stage != tt.stage || task.Next == "" {
			t.Errorf("%s/%s/%s: waiting %q stage %q next %q", tt.autonomy, tt.state, tt.size, task.Waiting, task.Stage, task.Next)
		}
	}
}
