package state

import "testing"

func TestCheck(t *testing.T) {
	tests := []struct {
		from, to, blockedFrom string
		ok                    bool
	}{
		{None, NeedsTriage, "", true},
		{None, Ready, "", true},
		{None, Planned, "", false},
		{NeedsTriage, Ready, "", true},
		{NeedsTriage, InProgress, "", false},
		{Ready, Planned, "", true},
		{Ready, InProgress, "", true},
		{Ready, Review, "", false},
		{Planned, InProgress, "", true},
		{Planned, Ready, "", true},
		{InProgress, Review, "", true},
		{InProgress, Planned, "", false},
		{Review, InProgress, "", true},
		{Review, Ready, "", false},
		{Review, Blocked, "", true},
		{None, Blocked, "", false},
		{Blocked, Review, Review, true},
		{Blocked, Ready, Review, false},
		{Blocked, Ready, "", true},
		{Ready, Ready, "", false},
		{Ready, "done", "", false},
	}
	for _, tt := range tests {
		err := Check(tt.from, tt.to, tt.blockedFrom)
		if (err == nil) != tt.ok {
			t.Errorf("Check(%q → %q, blockedFrom %q) = %v, want ok=%v", tt.from, tt.to, tt.blockedFrom, err, tt.ok)
		}
	}
}

func TestCurrent(t *testing.T) {
	s, err := Current([]string{"type:bug", "kiln:review", "size:S"})
	if err != nil || s != Review {
		t.Fatalf("got %q, %v", s, err)
	}
	if s, _ := Current([]string{"bug"}); s != None {
		t.Fatalf("want none, got %q", s)
	}
	if _, err := Current([]string{"kiln:ready", "kiln:planned"}); err == nil {
		t.Fatal("two kiln labels should be an error")
	}
}

func TestBlockedFrom(t *testing.T) {
	bodies := []string{"hi", "**Blocked** " + BlockedMarker(Planned), "later", "again " + BlockedMarker(Review)}
	if got := BlockedFrom(bodies); got != Review {
		t.Fatalf("got %q, want the latest marker", got)
	}
	if got := BlockedFrom([]string{"nothing"}); got != "" {
		t.Fatalf("got %q", got)
	}
}
