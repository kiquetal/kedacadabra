package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	namespace = "apps"
	yamlFile  = "keda-weekend-schedule.yaml"
)

// Status holds the current cluster state.
type Status struct {
	PauseSchedule  string
	ResumeSchedule string
	PausedReplicas string
	Replicas       string
	Error          string
}

func kubectl(args ...string) ([]byte, error) {
	return exec.Command("kubectl", args...).CombinedOutput()
}

// FetchStatus reads current CronJob schedules, ScaledObject annotation, and replica count.
func FetchStatus() Status {
	s := Status{PauseSchedule: "?", ResumeSchedule: "?", PausedReplicas: "?", Replicas: "?"}

	// CronJob schedules
	out, err := kubectl("get", "cronjob", "-n", namespace, "-o", "json")
	if err == nil {
		var result struct {
			Items []struct {
				Metadata struct{ Name string } `json:"metadata"`
				Spec     struct{ Schedule string } `json:"spec"`
			} `json:"items"`
		}
		if json.Unmarshal(out, &result) == nil {
			for _, item := range result.Items {
				switch item.Metadata.Name {
				case "keda-pause-weekend":
					s.PauseSchedule = item.Spec.Schedule
				case "keda-resume-monday":
					s.ResumeSchedule = item.Spec.Schedule
				}
			}
		}
	} else {
		s.Error = fmt.Sprintf("cronjobs: %s", strings.TrimSpace(string(out)))
		return s
	}

	// ScaledObject paused annotation
	out, err = kubectl("get", "scaledobject", "demo-app-scaler", "-n", namespace,
		"-o", "jsonpath={.metadata.annotations.autoscaling\\.keda\\.sh/paused-replicas}")
	if err == nil {
		v := strings.TrimSpace(string(out))
		if v == "" {
			s.PausedReplicas = "not paused"
		} else {
			s.PausedReplicas = v
		}
	}

	// Deployment replicas
	out, err = kubectl("get", "deployment", "demo-app", "-n", namespace,
		"-o", "jsonpath={.status.readyReplicas}/{.spec.replicas}")
	if err == nil {
		s.Replicas = strings.TrimSpace(string(out))
	}

	return s
}

var scheduleRe = regexp.MustCompile(`(?m)^(\s+schedule:\s*)"([^"]*)"`)

// ReplaceSchedules replaces the two schedule fields in the YAML content.
// The first match is the pause CronJob, the second is the resume CronJob.
func ReplaceSchedules(yamlContent, pauseCron, resumeCron string) string {
	count := 0
	return scheduleRe.ReplaceAllStringFunc(yamlContent, func(match string) string {
		count++
		sub := scheduleRe.FindStringSubmatch(match)
		switch count {
		case 1:
			return sub[1] + `"` + pauseCron + `"`
		case 2:
			return sub[1] + `"` + resumeCron + `"`
		}
		return match
	})
}

// ApplyCronSchedule updates the YAML file with new schedules and applies it.
func ApplyCronSchedule(pauseCron, resumeCron string) error {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", yamlFile, err)
	}

	updated := ReplaceSchedules(string(data), pauseCron, resumeCron)

	tmp, err := os.CreateTemp("", "keda-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(updated); err != nil {
		return err
	}
	tmp.Close()

	out, err := kubectl("apply", "-f", tmp.Name())
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}

	// Also update the local file
	return os.WriteFile(yamlFile, []byte(updated), 0644)
}
