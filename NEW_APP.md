# Adding a New App — Step by Step

This guide walks through adding `api-gateway` to kedacadabra, from cluster prerequisites to TUI management.

## Step 0 — Verify your cluster

Make sure KEDA and Prometheus are running:

```bash
kubectl get pods -n keda
# NAME                                      READY   STATUS
# keda-operator-...                         1/1     Running
# keda-metrics-apiserver-...                1/1     Running

kubectl get svc -n monitoring | grep prometheus
# prometheus-kube-prometheus-prometheus   ClusterIP   10.97.167.9   9090/TCP
```

The Prometheus address for ScaledObject triggers is:
```
http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090
```

## Step 1 — Create the Deployment

The container must expose metrics on an HTTP endpoint. We use `podinfo` which exposes
`http_requests_total` on port `9898` at `/metrics`.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: apps
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: ghcr.io/stefanprodan/podinfo:6.5.3
        ports:
        - containerPort: 9898
          name: http
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
EOF
```

Verify it's running and exposes metrics:

```bash
kubectl get deploy api-gateway -n apps
# NAME          READY   UP-TO-DATE   AVAILABLE
# api-gateway   1/1     1            1

kubectl exec -n apps deploy/api-gateway -- wget -qO- http://localhost:9898/metrics | grep http_requests_total
# http_requests_total{status="200"} 1
```

## Step 2 — Create the Service

The Service gives the Deployment a stable endpoint that Prometheus can scrape.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
  namespace: apps
  labels:
    app: api-gateway
spec:
  type: ClusterIP
  ports:
  - port: 9898
    targetPort: 9898
    protocol: TCP
    name: http
  selector:
    app: api-gateway
EOF
```

Verify:

```bash
kubectl get svc api-gateway -n apps
# NAME          TYPE        CLUSTER-IP       PORT(S)
# api-gateway   ClusterIP   10.101.107.77    9898/TCP
```

## Step 3 — Create the ServiceMonitor

The ServiceMonitor tells Prometheus to scrape the Service. The `release: prometheus`
label is required — it matches the Prometheus Operator's `serviceMonitorSelector`.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: api-gateway
  namespace: apps
  labels:
    app: api-gateway
    release: prometheus
spec:
  selector:
    matchLabels:
      app: api-gateway
  endpoints:
  - port: http
    interval: 15s
EOF
```

Verify Prometheus discovers the target (may take ~30s):

```bash
kubectl exec -n monitoring prometheus-prometheus-kube-prometheus-prometheus-0 -- \
  wget -qO- 'http://localhost:9090/api/v1/targets?state=active' | \
  python3 -c "
import json,sys
data = json.load(sys.stdin)
for t in data['data']['activeTargets']:
    if 'api-gateway' in t.get('labels',{}).get('job',''):
        print(f'job={t[\"labels\"][\"job\"]} health={t[\"health\"]}')
"
# job=api-gateway health=up
```

## Step 4 — Create the ScaledObject

The ScaledObject tells KEDA to autoscale the Deployment based on Prometheus metrics.
The query uses `namespace="apps"` and `pod=~"api-gateway.*"` to scope to this app only.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: api-gateway-scaler
  namespace: apps
spec:
  scaleTargetRef:
    name: api-gateway
  minReplicaCount: 1
  maxReplicaCount: 10
  pollingInterval: 30
  cooldownPeriod: 300
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090
      metricName: http_requests_rate
      query: sum(rate(http_requests_total{namespace="apps",pod=~"api-gateway.*"}[1m]))
      threshold: "10"
EOF
```

Verify:

```bash
kubectl get scaledobject api-gateway-scaler -n apps
# NAME                 SCALETARGETNAME   MIN   MAX   READY   ACTIVE   PAUSED   TRIGGERS
# api-gateway-scaler   api-gateway       1     10    True    False    False    prometheus
```

## Step 5 — Add to kedacadabra config

Edit `kedacadabra.yaml`:

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
    namespace: apps
    deployment: api-gateway
    scaledObject: api-gateway-scaler
    pauseCronJob: keda-pause-api-gateway
    resumeCronJob: keda-resume-api-gateway
    yamlFile: schedules/api-gateway.yaml
```

## Step 6 — Generate the schedule manifest

```bash
go run . generate
# Generating schedule manifests...
#   exists: schedules/demo-app.yaml
#   created: schedules/api-gateway.yaml
# Done. Apply with: kubectl apply -f schedules/<app>.yaml
```

This creates `schedules/api-gateway.yaml` with 5 resources:
- **ServiceAccount** `keda-cronjob-sa` — identity for CronJob pods
- **Role** `keda-cronjob-role` — permission to get/patch ScaledObjects
- **RoleBinding** `keda-cronjob-rolebinding` — binds Role to ServiceAccount
- **CronJob** `keda-pause-api-gateway` — annotates ScaledObject with `paused-replicas=0`
- **CronJob** `keda-resume-api-gateway` — removes the pause annotation

## Step 7 — Apply the schedule

```bash
kubectl apply -f schedules/api-gateway.yaml
# serviceaccount/keda-cronjob-sa unchanged
# role.rbac.authorization.k8s.io/keda-cronjob-role unchanged
# rolebinding.rbac.authorization.k8s.io/keda-cronjob-rolebinding unchanged
# cronjob.batch/keda-pause-api-gateway created
# cronjob.batch/keda-resume-api-gateway created
```

Note: the RBAC resources show `unchanged` because they're shared across apps in the same namespace.

## Step 8 — Verify everything

```bash
kubectl get cronjob -n apps
# NAME                      SCHEDULE       SUSPEND   ACTIVE   LAST SCHEDULE
# keda-pause-api-gateway    0 22 * * 5     False     0        <none>
# keda-pause-weekend        ...            False     0        ...
# keda-resume-api-gateway   0 6 * * 1      False     0        <none>
# keda-resume-monday        ...            False     0        ...
```

## Step 9 — Run the TUI

```bash
go run .
```

Use `←` / `→` (or `h` / `l`) to switch between `demo-app` and `api-gateway`.
The dashboard shows each app's current schedules, paused status, and replica count.

Use `F1` (relative mode) to test quickly — e.g., pause in 2 minutes, resume in 5 minutes.

## Summary of resources per app

| Resource | Purpose | Created by |
|---|---|---|
| Namespace | Workload boundary | You (pre-existing) |
| Deployment | The app to scale | You |
| Service | Stable endpoint for scraping | You |
| ServiceMonitor | Tells Prometheus to scrape the Service | You |
| ScaledObject | Tells KEDA how to autoscale | You |
| ServiceAccount | Identity for CronJob pods | `go run . generate` |
| Role | Permission to patch ScaledObjects | `go run . generate` |
| RoleBinding | Binds Role to ServiceAccount | `go run . generate` |
| Pause CronJob | Annotates ScaledObject → scale to zero | `go run . generate` |
| Resume CronJob | Removes annotation → restore autoscaling | `go run . generate` |
