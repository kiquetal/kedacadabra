package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Status holds the current cluster state for one app.
type Status struct {
	AppName        string
	Namespace      string
	PauseSchedule  string
	ResumeSchedule string
	PausedReplicas string
	Replicas       string
	Error          string
}

func kubectl(args ...string) ([]byte, error) {
	return exec.Command("kubectl", args...).CombinedOutput()
}

// FetchStatus reads current state for the given app.
func FetchStatus(app AppConfig) Status {
	s := Status{
		AppName: app.Name, Namespace: app.Namespace,
		PauseSchedule: "?", ResumeSchedule: "?", PausedReplicas: "?", Replicas: "?",
	}

	out, err := kubectl("get", "cronjob", "-n", app.Namespace, "-o", "json")
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
				case app.PauseCronJob:
					s.PauseSchedule = item.Spec.Schedule
				case app.ResumeCronJob:
					s.ResumeSchedule = item.Spec.Schedule
				}
			}
		}
	} else {
		s.Error = fmt.Sprintf("cronjobs: %s", strings.TrimSpace(string(out)))
		return s
	}

	out, err = kubectl("get", "scaledobject", app.ScaledObject, "-n", app.Namespace,
		"-o", "jsonpath={.metadata.annotations.autoscaling\\.keda\\.sh/paused-replicas}")
	if err == nil {
		v := strings.TrimSpace(string(out))
		if v == "" {
			s.PausedReplicas = "not paused"
		} else {
			s.PausedReplicas = v
		}
	}

	out, err = kubectl("get", "deployment", app.Deployment, "-n", app.Namespace,
		"-o", "jsonpath={.status.readyReplicas}/{.spec.replicas}")
	if err == nil {
		s.Replicas = strings.TrimSpace(string(out))
	}

	return s
}

var scheduleRe = regexp.MustCompile(`(?m)^(\s+schedule:\s*)"([^"]*)"`)

// ReplaceSchedules replaces the two schedule fields in the YAML content.
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

// ApplyCronSchedule updates the YAML file for the given app and applies it.
func ApplyCronSchedule(app AppConfig, pauseCron, resumeCron string) error {
	data, err := os.ReadFile(app.YAMLFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", app.YAMLFile, err)
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

	return os.WriteFile(app.YAMLFile, []byte(updated), 0644)
}
