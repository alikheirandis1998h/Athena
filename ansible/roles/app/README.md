# app

Deploys the sample app (`app/`) to the cluster with the Helm chart in `app/chart`. Runs on the first control-plane node.

Run it with `make app up APP_RELEASE=vX.Y.Z-release`. The release is a tag named `vMAJOR.MINOR.PATCH-release` on a commit that CI built. The Makefile resolves the tag to its commit and passes both to Ansible. The chart deploys the images tagged `api-vX.Y.Z-release` and `web-vX.Y.Z-release`, which CI publishes when the tag is pushed. If `APP_RELEASE` is not given, it defaults to the latest `v*-release` tag reachable from HEAD.

Create a release with `git tag -a vX.Y.Z-release <commit> -m "..."`. Push the tag when it is ready to share.

## What it does

- Generates the Postgres password and the two JWT secrets on the first run and stores them in `ansible/.secrets/app/` on the controller. Later runs reuse them. The directory is gitignored.
- Applies them as the `app-secrets` Secret in the `app` namespace. The Secret never goes into Helm values, and the rendered file is deleted from cp-1 right after it is applied.
- Installs or upgrades the `app` Helm release with `--wait`, then restarts the API and web pods so they pull the current images.

## Chart (app/chart)

- `app-postgres`: a StatefulSet on a 5 Gi Longhorn volume.
- `app-api`: the Go API on port 8080, served at `/health`.
- `app-web`: the Expo web build served by nginx on port 80.
- `app-migrate`: a post-install and post-upgrade Job that runs the SQL migrations. An init container copies the migration files out of the API image.
- `app`: the Ingress for the web host and the API host, using the `nginx` class.

## Access

- Web: `http://baseapp.local:30080`
- API: `http://api.baseapp.local:30080`

Port 30080 is the ingress NodePort. The deploy also writes a managed block to `/etc/hosts` on the controller, pointing both hostnames at `athena-worker-1`:

```
# BEGIN ANSIBLE athena app hosts
192.168.56.21 baseapp.local api.baseapp.local
# END ANSIBLE athena app hosts
```

That step needs sudo on the controller, so `make app up` asks for your password (once, shared with `make up`). Browsers on other machines need the same line added by hand.

## Known limits

- The web image bakes in `EXPO_PUBLIC_API_URL` at build time. Set the GitHub Actions variable of that name to `http://api.baseapp.local:30080` before the pipeline builds, or the web app calls `localhost`.
- Postgres and the migrate CLI come from Docker Hub, not `docker.novin.cloud`. Both are overridable in `app/chart/values.yaml`.
- There is no backup for Postgres yet. Longhorn backups are disabled, as for the rest of the cluster.
- The migrations run after the new API pods are ready, so a migration that changes the schema in a breaking way needs a separate rollout.
