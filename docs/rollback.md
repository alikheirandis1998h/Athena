# Rollback design

Status: designed. The rollback path below exists in the repository, but the procedure has not been run on the cluster as a rollback.

## What is rolled back

The unit of rollback is the app release: one `vX.Y.Z-release` tag. A release is four things, and each has its own store:

| Part | Where it lives | How it is kept for rollback |
|---|---|---|
| API and web images | Docker Hub `alikheirandis1998h/base`, tags `api-<tag>` and `web-<tag>` | Tags are immutable and never deleted by this repository |
| Chart and values | `app/chart/` in Git, plus the values Ansible renders at deploy time | Helm keeps each revision's manifests and values in the cluster |
| Secrets | `ansible/.secrets/app/` on the controller, applied as the `app-secrets` Secret | Not changed by a release, so a rollback reuses the current Secret |
| Database | PostgreSQL on a Longhorn volume, schema changed by the migrate Job | Not rolled back by default (see "Database" below) |

## Identifying the last healthy release

1. The last healthy release is the most recent tag whose deploy passed `make test`. Record each passing deploy in the release notes or a deploy log. This repository does not keep such a log yet.
2. Helm keeps the history: `helm history app -n app` lists each revision with its status and chart version. The revision whose status is `deployed` before the bad one is the candidate.
3. Confirm the candidate's images still exist on Docker Hub before rolling back to it.

## Triggers

A rollback starts when any of these hold after a deploy:

- `make test` fails on the new release and the cause is not a quick fix.
- The ingress 5xx rate is above the alert threshold in `docs/observability.md` for 10 minutes after the deploy.
- A Deployment does not reach its ready replica count within `helm upgrade --wait --timeout 10m`.

The operator decides. Nothing rolls back automatically.

## Procedure

Preferred path: redeploy the previous tag. This keeps Ansible and Helm in step, and it runs the migrate hook on the way back:

```
make app up APP_RELEASE=vX.Y.(Z-1)-release
make test
```

Emergency path: Helm rollback on the control plane, when `make app up` cannot run (for example, the controller is unavailable):

```
helm history app -n app
helm rollback app <revision> -n app --wait --timeout 10m
kubectl -n app rollout status deploy/app-api
kubectl -n app rollout status deploy/app-web
make test
```

`helm rollback` restores the manifests and values of the old revision. It does not run the migrate Job, because the Job is a `post-install,post-upgrade` hook and no rollback hook exists. The emergency path is therefore only safe when the schema has not changed between the two releases. Record the rollback in the incident notes and redeploy the release tag as soon as the controller is available, so Git and the cluster agree again.

## Who may roll back

- Any operator with access to the controller's kubeconfig can run either path. Today that is `admin.conf` (cluster-admin), so there is no separate rollback permission. A production design would give rollback its own role, bound to the Deployments and Helm release only.
- A rollback that changes the database schema needs a second person to review the plan.

## Database

Migrations are forward-only. The migrate Job runs `migrate up` and has no `down` step.

- Rollback is safe when the newer release only added tables, columns, or indexes that the older release ignores.
- Rollback is not safe when the newer release dropped or renamed a column the older release reads. In that case, roll forward with a fix instead.
- Production rule (not enforced here): use expand and contract. Release N adds the new column and writes both. Release N+1 switches reads. A later release drops the old column. Each step is then safe to roll back by one release.
- Restore from backup is the last resort for data damage. Backups are not configured (Longhorn backups are disabled, see the README).

## Verification after a rollback

- `make test` passes. It checks the running images are the expected immutable tags, the replicas are ready, and the application answers through ingress.
- The 5xx rate and the readiness of the pods return to the baseline in `docs/observability.md`.
- `helm history app -n app` shows the rollback as the latest revision.

## Failure of the rollback itself

- If the old release fails too, stop the rollback. Scale nothing down by hand; keep the current pods serving if they answer any traffic, and escalate.
- If the old images are missing from Docker Hub, the rollback cannot run. This is why tags are never deleted. Keep the last three release tags' images and record any manual deletion in the release notes.
- If `helm rollback` hangs, the `--timeout` expires and Helm reports the failed revision. Check `kubectl -n app describe pod` for the new pods, then rerun the preferred path.

## Gaps against production

- No deploy log or automatic record of the last healthy release.
- No automatic rollback on alert.
- Rollback permission is not separate from cluster-admin.
- No tested backup and restore for the database.
