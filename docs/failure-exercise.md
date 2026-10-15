# Failure exercises: API pod deletion and worker node failure

Run `make failure pod`, `make failure node`, or `make failure` for both (pod first). The playbooks are `ansible/playbooks/failure-pod.yaml` and `ansible/playbooks/failure-node.yaml`.

## Pod failure

Run `make failure pod`.

## Scenario

The exercise deletes one `app-api` pod while a client calls `/health` through the ingress once a second. It then checks that the API recovers within 120 seconds. This is a pod failure, not a node failure. It is the smallest real failure that exercises the probes, the Deployment, and the ingress routing.

## Expected behavior

- The Deployment creates a replacement pod right away.
- With one replica, no other pod can serve traffic, so the API is unavailable until the replacement pod passes its readiness probe.
- The ingress stops routing to the deleted pod as soon as it is removed from the endpoints.
- The API recovers without manual steps.

## Observed behavior

Two runs on the local cluster, on the image `api-99937c1`.

| Measure | 1 replica (before) | 2 replicas + PDB (after) |
|---|---|---|
| Pod deleted | `app-api-577d59dc88-tkxzn` | `app-api-86dbfdb858-9p7tv` |
| First failed sample | 0 s after the delete | none observed |
| Failed samples | 8 of 9 | 0 of 9 |
| Outage | about 9 s | none |
| Replacement pod | ready 1/1 | ready 2/2 |

With two replicas the ingress kept sending requests to the surviving pod, so the API did not go down.

The 1-replica run was done before the 2-replica change. The 2-replica run was done after `make app deploy`.

The delete used the normal graceful path. The events show `Killing: Stopping container api`, then the replacement pod's scheduling and start. The API's SIGTERM handler gives in-flight requests up to 10 seconds to finish.

## Detection

- The exercise's own sampler detects the failure from the outside, through the same path a user takes.
- `kubectl get events` for the deleted pod shows the kill and the replacement.
- `kubectl get deploy app-api` shows `readyReplicas` drop to 0 and come back to 1.

Nothing alerts on this. The outage is only visible to someone running the exercise or looking at the cluster by hand.

## User impact

About 9 seconds in which every request to `api.baseapp.local` fails, including sign-in and token refresh.

## Recovery mechanism

The Deployment controller replaces the pod. The readiness probe on `/health` (5 s period) keeps traffic away until the new pod is ready. The liveness probe is not involved, because the pod was deleted rather than hung.

## Remaining risks

- **One replica meant a full outage on every pod restart.** This was the main finding. The chart now sets `replicas: 2` and a PodDisruptionBudget with `minAvailable: 1`, and the 2-replica run showed no outage. Both replicas can still land on the same worker, because there is no anti-affinity.
- **Nothing alerts.** An outage like this is invisible until someone looks.
- **The exercise is not automated in CI.** It needs the live cluster, so it runs by hand.
- **The API has no `/version` endpoint.** The version is known only from the image tag.

## Production differences

- Two or more replicas spread across failure domains (nodes or zones), with anti-affinity and a PodDisruptionBudget. A single pod deletion would then cause no outage.
- Automatic alerting on the error rate at the ingress and on the readiness of the Deployment.
- Load balancing across ingress controller replicas, so one ingress pod can fail without an outage.
- The exercise repeated on a schedule against a staging environment, not run by hand on the lab cluster.

The playbook is safe to repeat. It deletes whichever `app-api` pod is first in the list, and it fails if the API does not recover within 120 seconds.

## Node failure

Run `make failure node`.

### Scenario

The exercise picks a worker that does not run `app-postgres` and that runs an API pod. It samples `/health` through the ingress on a surviving worker once a second, then powers the victim off with `virsh destroy`. This is a hard power-off, not a clean shutdown. It then waits for the cluster to mark the node NotReady and for the API to run back to full replicas on the surviving workers. Finally it powers the VM back on and waits for the node to report Ready.

### Expected behavior

- The node goes NotReady after the kubelet's heartbeat stops and the node-monitor grace period (40 s) passes.
- The API keeps serving from the surviving replica through the ingress, with few or no failed requests.
- Pods on the dead node are evicted after the default 300 s toleration and replaced on surviving workers.
- The worker rejoins after it is powered back on.

### Observed behavior

One run on the local cluster, on the images `api-v0.1.1-release` and `web-v0.1.1-release`. The victim was `athena-worker-3`; the probe went to `athena-worker-1`.

| Measure | Value |
|---|---|
| Node marked NotReady | 47 to 52 s after power-off |
| API back on surviving workers | 355 to 373 s after power-off |
| Worker rejoined | 374 to 392 s after power-off |
| Failed samples | 11 of 346 (first 2 s, last 44 s) and 29 of 698 (first 4 s, last 53 s) |

In both runs the API failed sampled requests starting a few seconds after the power-off and ending shortly after the node was marked NotReady. So the API was not fully available during the window, even with a surviving replica. The likely cause is that the dead pod stayed in the endpoint list until the node was marked NotReady, and the survivor's ingress path still routed some requests to it. This is not confirmed.

### Detection

- The sampler detects the failure from outside, through the same ingress path a user takes.
- `kubectl get nodes` shows the victim NotReady.
- `kubectl get pods -o wide` shows the evicted pods Terminating and the replacements on surviving workers.

Nothing alerts on this.

### User impact

Failed requests to `api.baseapp.local` for about 45 s after the power-off. Across the sampled window, 4% (29 of 698) and 3% (11 of 346) of requests failed.

### Recovery mechanism

Kubernetes marks the node NotReady and evicts its pods after 300 s. The Deployment creates replacement pods on the surviving workers, and the readiness probe keeps traffic away until they are ready. The Longhorn volume for `app-postgres` is on another worker and was unaffected. Once powered back on, the worker rejoins without manual steps.

### Remaining risks

- **No anti-affinity.** Both API replicas can land on the same worker, so one node failure can take out both.
- **The 300 s eviction delay is the main recovery limit.** Until then, Kubernetes keeps the dead pods in the endpoint list. Tuning the toleration would shorten the outage but makes flapping nodes more disruptive.
- **Longhorn briefly showed pods not Running after the worker returned.** `make test` failed once at the Longhorn check and passed on rerun. The exercise does not wait for Longhorn to settle.
- **The failure is simulated with `virsh destroy` on one VM,** not a real power loss or network partition.

### Production differences

- Spread replicas across nodes or zones with anti-affinity and a PodDisruptionBudget.
- Tune node-lifecycle timeouts for the failure domain, and use a load balancer in front of the ingress nodes that drops unhealthy nodes quickly.
- Alert on NotReady nodes and on ingress error rates.
- Run the exercise on a schedule in a staging environment.
