# Observability design

Status: designed, not implemented. The cluster has the raw signals (probes, events, pod status, container logs). There is no metrics store, dashboards, alerting, or trace collection. Metrics and alerting are optional bonus items in the assignment; this document says how they would be built.

## What exists today

| Signal | Source | How it is read |
|---|---|---|
| Node and cluster status | kubelet, control plane | `make k8s status`, `kubectl get nodes` |
| Pod readiness and restarts | kubelet | `kubectl get pods`, `make test` |
| Events | Kubernetes API | `kubectl get events`; `make failure` prints events for its exercises |
| Application logs | API (zerolog, JSON to stdout), web (nginx) | `kubectl logs` |
| Liveness and readiness | Probes in the chart | Pod status |
| Ingress reachability | `make test` | Curl through each worker's NodePort |

Logs are lost when a pod is deleted, because nothing collects them off the node. Events expire after about an hour.

## Signals to add

Four golden signals for each user-facing service, and the cluster health that the services depend on.

| Area | Signal | Source |
|---|---|---|
| Traffic | Requests per second per route and host | F5 NGINX Ingress metrics |
| Errors | 5xx ratio per host | F5 NGINX Ingress metrics |
| Latency | p50, p95, p99 per host | F5 NGINX Ingress metrics |
| Saturation | CPU and memory per pod, against requests and limits | cAdvisor through kubelet, kube-state-metrics |
| Availability | Ready replicas per Deployment, PDB disruptions allowed | kube-state-metrics |
| Restarts | Container restart count | kube-state-metrics |
| Nodes | Ready state, memory and disk pressure, pod count | kube-state-metrics, node-exporter |
| Storage | Longhorn volume state (healthy, degraded), replica count, free space on the Longhorn disks | Longhorn metrics |
| Database | Connections, transactions, replication lag (none here), disk | postgres-exporter (not yet added) |
| Network | Cilium agent and drop counters | Cilium metrics; Hubble for flow visibility |
| Application | Requests by route and status; the version the app runs | Needs `/metrics` in the API (not built). Version is the image tag today |

## Collection

- Prometheus scrapes the sources above through ServiceMonitors. Retention 15 days on local disk, or remote-write to long-term storage in production.
- Node-exporter runs on each node. kube-state-metrics runs once per cluster.
- Logs: a node agent (Promtail or Vector) ships container logs to Loki, with a 14-day retention. Logs carry the pod, namespace, and container labels so a log line can be matched to a metric.
- Traces: not planned for this scope. The API has no tracing hook. Add OpenTelemetry only when a dependency needs it.

## Dashboards

Each dashboard answers one question.

1. **Service health**: request rate, 5xx ratio, p95 latency, and ready replicas for `app-api` and `app-web`. Answers "is the app up for users?"
2. **Cluster health**: node Ready state, memory and disk pressure, pod count per node, Cilium and Longhorn state. Answers "is the platform healthy?"
3. **Deploy view**: the release tag running, the replica rollout progress, and the restart count since the deploy. Answers "did the last deploy go well?"
4. **Storage**: Longhorn volume health and free space on the PostgreSQL volume. Answers "is the database disk at risk?"

## Alerts and SLOs

Service objective (proposed): 99.5% of requests through ingress return a non-5xx response over 30 days. That is about 3.6 hours of error budget per month.

| Alert | Condition | Severity | Why |
|---|---|---|---|
| High 5xx ratio | 5xx ratio above 1% for 5 minutes | Page | User-visible failure |
| Burn rate (fast) | Error budget burn 14x over 1 hour | Page | Budget gone in about two days |
| Burn rate (slow) | Error budget burn 3x over 6 hours | Ticket | Slow leak |
| Replicas unavailable | Ready replicas below desired for 5 minutes | Page | Less capacity, or the PDB blocks maintenance |
| Crash loop | Container restarts more than 3 times in 15 minutes | Ticket | Bad release or bad config |
| Node not ready | Node NotReady for 2 minutes | Page | The failure exercise showed 47–52 s of NotReady and about 5 min before eviction |
| Node pressure | MemoryPressure or DiskPressure for 5 minutes | Ticket | Evictions are coming |
| Volume degraded | Longhorn volume degraded for 10 minutes | Page | Lost replica; the PostgreSQL volume has 3 replicas |
| Disk low | Free space below 15% on a Longhorn disk | Ticket | Volume cannot grow |
| Deploy stuck | Rollout not complete after the 10 m Helm timeout | Ticket | Rollback candidate (`docs/rollback.md`) |
| Watchdog | Always firing | Dead-man switch to an external service | Detects that the alerting path is broken |

Alerts link to the matching dashboard and to the runbook section in `docs/rollback.md` or the failure exercise notes.

## Access

- Grafana with SSO. No shared admin account.
- Read-only dashboards for most people. Alert silencing needs the on-call role.
- Prometheus and Alertmanager are reachable only inside the cluster, through port-forward or an internal ingress.

## Observability failure

- If Prometheus stops, the Watchdog alert stops arriving at the external service, which pages on silence.
- If Alertmanager fails, the same Watchdog path detects it.
- If Loki fails, logs are lost for that period. Metrics and alerts still work. Keep Loki's own disk on a separate volume so a full log store cannot fill the node disk.
- The observability stack runs on the same nodes as the app. Put its pods on a node pool that the app cannot fill, or give them resource requests so they are not evicted first.

## Local scope

- Not built here: the metrics stack, Loki, the dashboards, the alerts, and the `/metrics` endpoint on the API.
- Built and checked by hand: pod status, events, logs, probes, and `make test`.
- The first build step would be kube-prometheus-stack (Prometheus, Alertmanager, Grafana, kube-state-metrics, node-exporter), then the ingress metrics, then Loki.
