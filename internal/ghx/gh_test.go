package ghx

import "testing"

func TestSummarize(t *testing.T) {
	s := Summarize([]Check{
		{Typename: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		{Typename: "CheckRun", Name: "skip", Status: "COMPLETED", Conclusion: "SKIPPED"},
		{Typename: "CheckRun", Name: "lint", Status: "COMPLETED", Conclusion: "FAILURE"},
		{Typename: "CheckRun", Name: "e2e", Status: "IN_PROGRESS"},
		{Typename: "StatusContext", Context: "deploy", State: "PENDING"},
		{Typename: "StatusContext", Context: "cla", State: "SUCCESS"},
		{Typename: "StatusContext", Context: "sec", State: "ERROR"},
	})
	if s.Total != 7 || s.Passed != 3 || s.Failing != 2 || s.Pending != 2 || s.Failed[0] != "lint" || s.Failed[1] != "sec" {
		t.Fatalf("%+v", s)
	}
}
