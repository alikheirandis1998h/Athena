# Failure exercise: API pod deletion

Run with `make failure`. The playbook is `ansible/playbooks/failure-exercise.yaml`.

## Scenario

The exercise deletes one `app-api` pod while a client calls `/health` through the ingress once a second. It then checks that the API recovers within 120 seconds. This is a pod failure, not a node failure. It is the smallest real failure that exercises the probes, the Deployment, and the ingress routing.

## Expected behavior

- The Deployment creates a replacement pod right away.
- With one replica, no other pod can serve traffic, so the API is unavailable until the replacement pod passes its readiness probe.
- The ingress stops routing to the deleted pod as soon as it is removed from the endpoints.
- The API recovers without manual steps.

## Observed behavior

One run on the local cluster, on the release image `api-99937c1`:

| Measure | Value |
|---|---|
| Pod deleted | `app-api-577d59dc88-tkxzn` (on `athena-worker-2`) |
| First failed sample | 0 s after the delete |
| Failed samples | 8 of 9 |
| Outage | about 9 s (first failure to the first 200 after it) |
| Replacement pod | `app-api-577d59dc88-mdw98`, ready 1/1 |
| Recovery limit | 120 s (passed) |

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

- **One replica means a full outage on every pod restart.** This is the main finding. Any rollout, eviction, or node drain has the same effect. The fix is `replicas: 2` or more, plus a PodDisruptionBudget with `minAvailable: 1`. Both are not yet in the chart.
- **Nothing alerts.** An outage like this is invisible until someone looks.
- **The exercise is not automated in CI.** It needs the live cluster, so it runs by hand.
- **The API has no `/version` endpoint.** The version is known only from the image tag.

## Production differences

- Two or more replicas spread across failure domains (nodes or zones), with anti-affinity and a PodDisruptionBudget. A single pod deletion would then cause no outage.
- Automatic alerting on the error rate at the ingress and on the readiness of the Deployment.
- Load balancing across ingress controller replicas, so one ingress pod can fail without an outage.
- The exercise repeated on a schedule against a staging environment, not run by hand on the lab cluster.

## Re-running

```
make failure
```

The playbook is safe to repeat. It deletes whichever `app-api` pod is first in the list, and it fails if the API does not recover within 120 seconds.
