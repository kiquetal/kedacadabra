# kedacadabra ⚡

TUI to manage KEDA scale-to-zero schedules on Kubernetes. View cluster state and update CronJob schedules interactively.

## Prerequisites

- Go 1.25+
- `kubectl` configured and pointing at your cluster
- [KEDA](https://keda.sh) installed on the cluster

## Setup the cluster resources

```bash
kubectl apply -f namespace.yaml
kubectl apply -f demo-app.yaml
kubectl apply -f keda-scaledobject-prometheus.yaml
kubectl apply -f keda-weekend-schedule.yaml
```

This creates:
- `apps` namespace
- `demo-app` Deployment + Service + ServiceMonitor
- `demo-app-scaler` ScaledObject (Prometheus trigger)
- `keda-pause-weekend` CronJob — annotates ScaledObject to scale to zero
- `keda-resume-monday` CronJob — removes the pause annotation

## Run the TUI

```bash
go run .
```

Or build and run:

```bash
go build -o kedacadabra .
./kedacadabra
```

## Keybindings

| Key | Action |
|---|---|
| `F1` | Relative mode — set pause/resume as minutes from now |
| `F2` | Absolute mode — set weekday/hour/minute for each CronJob |
| `Tab` / `Shift+Tab` | Navigate between input fields |
| `Enter` | Apply schedule (updates YAML + `kubectl apply`) |
| `r` | Refresh cluster status |
| `q` / `Ctrl+C` | Quit |

## Modes

**Relative** — For testing. Enter minutes from now for pause and resume. E.g., "2" and "10" means pause in 2 minutes, resume in 10 minutes. Generates a one-shot cron expression for that exact time.

**Absolute** — For production schedules. Enter weekday (0=Sun, 1=Mon, ..., 6=Sat), hour, and minute for each CronJob. E.g., day=5 hour=22 min=0 for Friday 22:00 UTC.

## How it works

The TUI reads cluster state via `kubectl get` (CronJob schedules, ScaledObject paused annotation, deployment replicas). When you apply, it:

1. Reads `keda-weekend-schedule.yaml`
2. Replaces the `schedule` fields with your new cron expressions
3. Writes to a temp file and runs `kubectl apply -f`
4. Updates the local YAML file to keep it in sync
