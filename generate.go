package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const scheduleTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: keda-cronjob-sa
  namespace: {{.Namespace}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: keda-cronjob-role
  namespace: {{.Namespace}}
rules:
- apiGroups: ["keda.sh"]
  resources: ["scaledobjects"]
  verbs: ["get", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: keda-cronjob-rolebinding
  namespace: {{.Namespace}}
subjects:
- kind: ServiceAccount
  name: keda-cronjob-sa
  namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: keda-cronjob-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{.PauseCronJob}}
  namespace: {{.Namespace}}
spec:
  schedule: "0 22 * * 5"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: keda-cronjob-sa
          restartPolicy: OnFailure
          containers:
          - name: kubectl
            image: registry.k8s.io/kubectl:v1.32.0
            command:
            - kubectl
            - annotate
            - scaledobject
            - {{.ScaledObject}}
            - -n
            - {{.Namespace}}
            - autoscaling.keda.sh/paused-replicas=0
            - --overwrite
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: {{.ResumeCronJob}}
  namespace: {{.Namespace}}
spec:
  schedule: "0 6 * * 1"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: keda-cronjob-sa
          restartPolicy: OnFailure
          containers:
          - name: kubectl
            image: registry.k8s.io/kubectl:v1.32.0
            command:
            - kubectl
            - annotate
            - scaledobject
            - {{.ScaledObject}}
            - -n
            - {{.Namespace}}
            - autoscaling.keda.sh/paused-replicas-
            - --overwrite
`

// GenerateScheduleYAML creates the schedule YAML file for an app config.
func GenerateScheduleYAML(app AppConfig) error {
	tmpl, err := template.New("schedule").Parse(scheduleTemplate)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(app.YAMLFile), 0755); err != nil {
		return err
	}

	f, err := os.Create(app.YAMLFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, app)
}

// GenerateAllSchedules generates YAML for all apps that don't have a file yet.
func GenerateAllSchedules(cfg Config) error {
	for _, app := range cfg.Apps {
		if _, err := os.Stat(app.YAMLFile); err == nil {
			fmt.Printf("  exists: %s\n", app.YAMLFile)
			continue
		}
		if err := GenerateScheduleYAML(app); err != nil {
			return fmt.Errorf("generate %s: %w", app.Name, err)
		}
		fmt.Printf("  created: %s\n", app.YAMLFile)
	}
	return nil
}
