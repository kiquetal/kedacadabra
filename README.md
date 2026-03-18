# kedacadabra ⚡

TUI to manage KEDA scale-to-zero schedules on Kubernetes. View cluster state and update CronJob schedules interactively. Supports multiple apps across namespaces.

## Prerequisites

- Go 1.25+
- `kubectl` configured and pointing at your cluster
- [KEDA](https://keda.sh) installed on the cluster

## Quick start

### 1. Configure your apps

Edit `kedacadabra.yaml` to define which apps to manage:

```yaml
apps:
  - name: demo-app
    namespace: apps
    deployment: demo-app
    scaledObject: demo-app-scaler
    pauseCronJob: keda-pause-weekend
    resumeCronJob: keda-resume-monday
    yamlFile: schedules/demo-app.yaml

  - name: api-gateway
    namespace: production
    deployment: api-gateway
    scaledObject: api-gateway-scaler
    pauseCronJob: keda-pause-api-gateway
    resumeCronJob: keda-resume-api-gateway
    yamlFile: schedules/api-gateway.yaml
```

### 2. Generate the schedule manifests

```bash
go run . generate
```

This creates a YAML file per app under `schedules/` with:
- ServiceAccount + Role + RoleBinding (RBAC for CronJobs to patch ScaledObjects)
- Pause CronJob (annotates ScaledObject with `paused-replicas=0`)
- Resume CronJob (removes the pause annotation)

### 3. Apply to your cluster

```bash
kubectl apply -f schedules/demo-app.yaml
kubectl apply -f schedules/api-gateway.yaml
```

### 4. Run the TUI

```bash
go run .
```

## Adding a new app

1. Add an entry to `kedacadabra.yaml`:

```yaml
  - name: my-service
    namespace: my-namespace
    deployment: my-service
    scaledObject: my-service-scaler
    pauseCronJob: keda-pause-my-service
    resumeCronJob: keda-resume-my-service
    yamlFile: schedules/my-service.yaml
```

2. Generate and apply:

```bash
go run . generate
kubectl apply -f schedules/my-service.yaml
```

3. Run the TUI — your new app appears in the app selector.

## Verify the build

```bash
# Run tests
go test -v ./...

# Build the binary
go build -o kedacadabra .

# Check it
file kedacadabra
# → ELF 64-bit LSB executable, x86-64 ...

# Inspect embedded dependencies
go version -m kedacadabra
```

## Keybindings

| Key | Action |
|---|---|
| `→` / `l` | Next app |
| `←` / `h` | Previous app |
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

1. Reads the app's schedule YAML from `schedules/<app>.yaml`
2. Replaces the `schedule` fields with your new cron expressions
3. Writes to a temp file and runs `kubectl apply -f`
4. Updates the local YAML file to keep it in sync

## Project structure

```
kedacadabra.yaml          # Config: list of apps to manage
schedules/                # Generated schedule manifests (one per app)
  demo-app.yaml
main.go                   # Entry point: TUI or generate subcommand
config.go                 # Config file loading
generate.go               # Template-based manifest generation
kube.go                   # kubectl interactions (fetch status, apply)
cron.go                   # Time-to-cron conversion
model.go                  # Bubble Tea model (state, update, keybindings)
views.go                  # Bubble Tea view (rendering)
```
