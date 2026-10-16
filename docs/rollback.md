# Rollback design

Status: partly implemented. The current path (`make app up` with an older tag, or `helm rollback` on the controller) exists in the repository but has not been run on the cluster as a rollback. The GitOps path is designed in `docs/ci-cd.md` and not built.

This document covers two paths:

- **Current:** the release is deployed by `make app up` from this repository, and rolled back with the procedure under "Current procedure".
- **Target:** the release is deployed by Argo CD from `athena-gitops`, and rolled back by a Git revert. See "Target procedure". The target is defined in `docs/ci-cd.md`, Part 2.

## What is rolled back

The unit of rollback is the app release: one `vX.Y.Z-release` tag. A release is four things, and each has its own store:

| Part | Current location | Target location | How it is kept for rollback |
|---|---|---|---|
| API and web images | Docker Hub `alikheirandis1998h/base`, tags `api-<tag>` and `web-<tag>` | GitLab Container Registry, `backend:<tag>` and `web:<tag>` | Tags are never moved or deleted |
| Chart and values | `app/chart/` in Git, plus the values Ansible renders at deploy time | `charts/athena/` and `environments/<env>/values.yaml` in `athena-gitops` | Git history of the repository; the cluster keeps Helm revisions in the current path |
| Secrets | `ansible/.secrets/app/` on the controller, applied as the `app-secrets` Secret | Per cluster, not in Git | Not changed by a release, so a rollback reuses the current Secret |
| Database | PostgreSQL on a Longhorn volume, schema changed by the migrate Job | PostgreSQL per cluster (production and beta are separate) | Not rolled back by default (see "Database" below) |

## Identifying the last healthy release

Current:

1. The last healthy release is the most recent tag whose deploy passed `make test`. Record each passing deploy in the release notes or a deploy log. This repository does not keep such a log yet.
2. Helm keeps the history: `helm history app -n app` lists each revision with its status and chart version. The revision whose status is `deployed` before the bad one is the candidate.
3. Confirm the candidate's images still exist in the registry before rolling back to it.

Target:

1. The last healthy release is the previous merged production MR in `athena-gitops` that passed its pipeline, and the tag it set.
2. `git log -- environments/production/values.yaml` lists every production change, with its MR.
3. Argo CD's sync history for the `production` Application shows which revision was synced and whether it became healthy.

## Triggers

A rollback starts when any of these hold after a deploy:

- `make test` fails on the new release, or the release fails its post-deploy checks in the target path, and the cause is not a quick fix.
- The ingress 5xx rate is above the alert threshold in `docs/observability.md` for 10 minutes after the deploy.
- A Deployment does not reach its ready replica count within the rollout timeout (`helm upgrade --wait --timeout 10m` in the current path, the Argo CD health check in the target path).

The operator decides. Nothing rolls back automatically.

## Current procedure

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

## Target procedure

The target has no `make app up` and no operator-run `helm` for production. The rollback is a Git change:

1. Open an MR in `athena-gitops` that sets `environments/production/values.yaml` back to the previous `vX.Y.(Z-1)-release` tag. Use `git revert` on the commit that moved the tag, or set the tag by hand if the revert conflicts.
2. The GitOps pipeline lints and renders the change. A reviewer approves the MR. Merging it is the approval, the same as a forward release.
3. Argo CD syncs the `production` Application. The `PreSync` migrate hook runs first, then the Deployments change to the old tag.
4. Argo CD reports the Application healthy. Run `make test` against production, or its equivalent in the target design.

Beta has its own procedure, with the same steps against `environments/beta/values.yaml` and the beta cluster's Argo CD. A beta rollback has no approval gate beyond the MR review.

The `PreSync` migrate hook runs on the rollback sync too. It runs `migrate up`, which does nothing when the schema is already at the head. It does not undo a schema change. The "Database" rules below still apply.

## Who may roll back

Current:

- Any operator with access to the controller's kubeconfig can run either path. Today that is `admin.conf` (cluster-admin), so there is no separate rollback permission.
- A rollback that changes the database schema needs a second person to review the plan.

Target:

- Any person with merge rights on `athena-gitops` can roll back production by merging a revert MR, subject to the reviewer approval. Creating a release tag is limited to maintainers.
- Argo CD applies changes only within its namespace-scoped `AppProject`. It does not hold cluster-admin.
- A rollback that changes the database schema needs a second reviewer on the MR.

## Database

Migrations are forward-only. The migrate Job runs `migrate up` and has no `down` step.

- Rollback is safe when the newer release only added tables, columns, or indexes that the older release ignores.
- Rollback is not safe when the newer release dropped or renamed a column the older release reads. In that case, roll forward with a fix instead.
- Production rule (not enforced here): use expand and contract. Release N adds the new column and writes both. Release N+1 switches reads. A later release drops the old column. Each step is then safe to roll back by one release.
- Restore from backup is the last resort for data damage. Backups are not configured (Longhorn backups are disabled, see the README). The target has a separate PostgreSQL for beta, which is never restored from production data.

## Verification after a rollback

- `make test` passes, or the target's equivalent check passes. It checks the running images are the expected immutable tags, the replicas are ready, and the application answers through ingress.
- The 5xx rate and the readiness of the pods return to the baseline in `docs/observability.md`.
- Current: `helm history app -n app` shows the rollback as the latest revision. Target: the Argo CD Application shows the reverted revision as synced and healthy.

## Failure of the rollback itself

- If the old release fails too, stop the rollback. Scale nothing down by hand; keep the current pods serving if they answer any traffic, and escalate.
- If the old images are missing from the registry, the rollback cannot run. This is why tags are never deleted. Keep the last three release tags' images and record any manual deletion in the release notes.
- Current: if `helm rollback` hangs, the `--timeout` expires and Helm reports the failed revision. Check `kubectl -n app describe pod` for the new pods, then rerun the preferred path.
- Target: if the Argo CD sync stalls, check the Application's events and the `PreSync` Job logs before reverting the revert. Do not edit the cluster by hand to unstick a sync.

## Gaps against production

- No deploy log or automatic record of the last healthy release in the current path.
- No automatic rollback on alert.
- Current rollback permission is not separate from cluster-admin.
- No tested backup and restore for the database.
- The target path is not built. The GitOps repository, Argo CD, and the protected production environment do not exist yet.
