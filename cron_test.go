package main

import (
	"testing"
	"time"
	"fmt"
)

func TestRelativeToCron(t *testing.T) {
	got := RelativeToCron(5)
	now := time.Now().UTC().Add(5 * time.Minute)
	want := fmt.Sprintf("%d %d %d %d *", now.Minute(), now.Hour(), now.Day(), int(now.Month()))
	if got != want {
		t.Errorf("RelativeToCron(5) = %q, want %q", got, want)
	}
}

func TestAbsoluteToCron(t *testing.T) {
	tests := []struct {
		weekday, hour, minute int
		want                  string
	}{
		{5, 22, 0, "0 22 * * 5"},
		{1, 6, 0, "0 6 * * 1"},
		{0, 0, 30, "30 0 * * 0"},
	}
	for _, tt := range tests {
		got := AbsoluteToCron(tt.weekday, tt.hour, tt.minute)
		if got != tt.want {
			t.Errorf("AbsoluteToCron(%d,%d,%d) = %q, want %q",
				tt.weekday, tt.hour, tt.minute, got, tt.want)
		}
	}
}

func TestReplaceSchedules(t *testing.T) {
	yaml := `spec:
  schedule: "0 22 * * 5"
  jobTemplate:
---
spec:
  schedule: "0 6 * * 1"
  jobTemplate:`

	got := ReplaceSchedules(yaml, "30 18 * * 3", "0 19 * * 3")
	want1 := `schedule: "30 18 * * 3"`
	want2 := `schedule: "0 19 * * 3"`
	if !contains(got, want1) || !contains(got, want2) {
		t.Errorf("ReplaceSchedules failed.\nGot:\n%s", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
