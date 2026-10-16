# CI/CD

This document has two parts. Part 1 describes what is built today. Part 2 is the target design, which is not built yet.

- Part 1: GitHub Actions in this repository (`.github/workflows/ci-cd.yml`). It builds, tests, and publishes images. Deploy is manual.
- Part 2: GitLab CI with split repositories, GitOps, and Argo CD. The production update and the beta convention are defined here.

---

## Part 1: Implemented

Status: implemented for build, test, and publish. Promotion, approval, scanning, and deploy-permission separation are not built.

### Pipeline

```
push / pull request
      |
      v
backend-test ----+
  go vet, go test
                 +--> images --> (release tag or main) push to Docker Hub
web-check -------+       |
  npm ci, typecheck      v
                    api-<sha>, web-<sha>        (every publishing run)
                    api-<tag>, web-<tag>        (release tags)
                    api-latest, web-latest      (main only)
                                |
                                v
                      make app up APP_RELEASE=vX.Y.Z-release   (operator, not CI)
                                |
                                v
                      make test                                 (operator, on the cluster)
```

Triggers: pushes to `main`, pushes of `v*-release` tags, pull requests (build only, no push), and manual dispatch. `concurrency` cancels a superseded run on the same ref.

### Stages and what they guarantee

| Stage | Implemented | Guarantee | Gap |
|---|---|---|---|
| Backend checks | `go vet`, `go test` | Code compiles and vet passes | The backend has no test files, so `go test` runs no tests |
| Web checks | `npm ci`, `npm run typecheck` | Dependencies resolve from the lock file; types check | No unit or UI tests |
| Image build | Buildx, build-push, one image per app | The image is built from the commit that passed the checks | Build is not reproducible byte-for-byte |
| Image publish | Only on `main` and `v*-release` | Pull requests cannot push | Publish happens before any cluster test |
| Tagging | `api-<40-hex sha>`, `api-<release tag>`, `api-latest` on `main`; same for `web-` | Each image traces to a commit and, for releases, a tag | `latest` is kept only for `main`; releases are not checked for a signed tag |
| Deploy | `make app up` (Ansible + Helm) | The release tag resolves to a commit, and the chart uses the tag | Deploy is a manual step, run by an operator |
| Verify | `make test` | Pods run the expected images, replicas are ready, the app answers | Runs by hand, after the deploy |

### Promotion without rebuilding

Today, a release tag causes CI to build the images again from the tagged commit. This is a second build of the same source. The release image is not the one that passed the `main` checks byte-for-byte. A release tag must point to a commit that CI already built and tested. `make app up` checks the tag's commit and fails without an image.

### Credentials and permissions

- CI has `contents: read` and only the Docker Hub token as a secret. It cannot write to the repository or to the cluster.
- The deploy uses the controller's kubeconfig (`admin.conf`, cluster-admin), run by an operator. CI has no path to the cluster. This keeps CI off the cluster, but the deploy identity is far broader than needed.

### Gaps in Part 1

- No scanning, signing, or SBOM in the pipeline.
- No environments or approvals.
- Deploy identity is cluster-admin, and the deploy is manual.
- Release images are built again from the tag rather than promoted by digest.
- The backend has no tests, so the checks pass without testing behavior.

---

## Part 2: Target design (GitLab CI, GitOps, Argo CD)

Status: designed, not built.

### Repositories

The app is split into four repositories, so each one has its own pipeline, owners, and permissions.

| Repository | Contents | Pipeline produces |
|---|---|---|
| `athena-backend` | Go API (`app/backend`) | `backend` image |
| `athena-web` | Web app (`app`, excluding the backend) | `web` image |
| `athena-gitops` | Helm chart (`charts/athena`), Argo CD `Application`s, per-environment values | Commits that change deployed image tags |
| `athena-platform` | This repository: `vm/`, `ansible/`, `docs/`, `Makefile` | Nothing deployed by CI. It stays the host and cluster bootstrap |

The chart moves to `athena-gitops` because Argo CD needs one place that describes the desired state of every environment. The chart is versioned with the environment values that pin it.

### Registry

Images go to the GitLab Container Registry of the group:

```
registry.<gitlab-host>/athena/backend:<tag>
registry.<gitlab-host>/athena/web:<tag>
```

Tag rules:

| Tag | Set by | Meaning |
|---|---|---|
| `<40-hex sha>` | `main` pipeline | Built from a commit that passed the full checks on `main` |
| `vX.Y.Z-release` | release tag pipeline | Production release. Points to the same digest as the `<sha>` tag |
| `vX.Y.Z-beta.N` or any other tag | tag pipeline | A pre-release or test build. Deployable to beta only |

Images are never tagged `latest`, and a tag is never moved to a different digest.

### Merge request pipelines

An MR pipeline runs only the jobs whose scope changed. Scope comes from `rules: changes:` on the paths of each job. A docs-only change runs no build.

Backend (`athena-backend/.gitlab-ci.yml`):

| Changed path | Jobs |
|---|---|
| `**/*.go`, `go.mod`, `go.sum` | `go vet`, `golangci-lint`, `go test -race`, `govulncheck` |
| `migrations/**`, `queries/**`, `sqlc.yaml` | `sqlc diff`, and a migration run against a throwaway PostgreSQL service |
| `Dockerfile`, `**/*.go`, `go.mod`, `go.sum` | Build the image without pushing. Trivy scan |

Web (`athena-web/.gitlab-ci.yml`):

| Changed path | Jobs |
|---|---|
| `src/**`, `routes/**`, `package*.json`, `tsconfig.json`, `babel.config.js`, `app.config.ts` | `npm ci`, `npm run typecheck`, unit tests, `npm audit` |
| `Dockerfile`, `nginx.conf`, `src/**`, `package*.json` | Build the image without pushing. Trivy scan |

GitOps (`athena-gitops/.gitlab-ci.yml`):

| Changed path | Jobs |
|---|---|
| `charts/**` | `helm lint`, `kubeconform` on the rendered output |
| `environments/**` | `helm template` per environment, `kubeconform`, Argo CD diff against the live app |

Platform (`athena-platform/.gitlab-ci.yml`):

| Changed path | Jobs |
|---|---|
| `ansible/**` | `ansible-lint`, syntax check of every playbook |
| `vm/**` | Vagrantfile syntax check |

MR rules:
- Merge only when the pipeline passes and the MR has one approval from a code owner.
- Pipelines on MRs do not push images and do not change the GitOps repo.

### Pipeline on `main`

A commit on `main` runs every job for the repository, not only the changed scope. This catches a test that depends on code changed elsewhere. Build jobs run for the changed scope.

Backend, on `main`:
1. Run every backend job, including the full test suite and `govulncheck`.
2. Build `backend:<sha>` and push it to the registry.
3. Trivy scan the pushed image. Fail on high or critical findings that have a fix.
4. Record the digest as a pipeline artifact, so a later release can reuse it.

Web, on `main`: the same pattern for `web:<sha>`.

GitOps, on `main`: every chart and environment check runs. No image is built here.

The `main` pipeline does not deploy. A commit on `main` is a candidate. It reaches beta or production only through a tag.

### Release tag pipeline (`vX.Y.Z-release`)

Tags are protected in GitLab. Only maintainers can create them.

1. Check that the tagged commit is on `main` and that `backend:<sha>` and `web:<sha>` exist. If either is missing, the pipeline fails. This is the rule that the release is built from a commit CI already tested.
2. Promote by digest: tag the `<sha>` digest as `vX.Y.Z-release` in the registry. No rebuild happens.
3. Run the repository's full test job again against the tagged commit.
4. Open a merge request in `athena-gitops` that sets the production image tags in `environments/production/values.yaml` to `vX.Y.Z-release`. The MR has the pipeline output and the image digests in its description.
5. A reviewer approves. Merging the MR is the production approval.
6. Argo CD sees the merge and syncs the `production` Application.

Production updates go through GitOps only. Nothing runs `helm upgrade` by hand, and no CI job has cluster credentials.

### Beta environment convention

Beta lets you run any tag before it goes to production. Beta is for testing, not for production traffic.

Tag names:

| Tag pattern | Example | Use |
|---|---|---|
| `vX.Y.Z-release` | `v0.1.2-release` | Production candidate. Deployable to beta and production |
| `vX.Y.Z-beta.N` | `v0.2.0-beta.1` | Pre-release for beta testing |
| Any other `v*` tag | `v0.2.0-fix-login` | Test build. Deployable to beta |

Rules:
- Beta is deployed by a manual job named `deploy:beta` in the `athena-gitops` pipeline. It is not automatic on tag push. Every beta deploy is a deliberate action.
- The job takes a required variable `TAG`. It fails unless both images exist for that tag in the registry.
- The job writes the tag to `environments/beta/values.yaml` on a branch and opens an MR. Beta is updated when that MR is merged, the same way as production. This keeps every running version in Git.
- Only one beta deploy runs at a time, using a GitLab `resource_group: beta`.
- Beta runs on its own cluster, separate from production. It has its own Argo CD, its own ingress, and its own PostgreSQL. The app namespace is the same name on both clusters.
- Beta data is never restored from production data.

A release tag can be deployed to beta before production. This is the usual order: deploy `vX.Y.Z-release` to beta, test it, then merge the production MR.

### GitOps and Argo CD

Layout of `athena-gitops`:

```
charts/athena/                  Helm chart (moved from app/chart)
environments/beta/values.yaml   image tags and beta settings
environments/production/values.yaml
apps/beta.yaml                  Argo CD Application for beta
apps/production.yaml            Argo CD Application for production
```

Each cluster runs its own Argo CD instance, installed on that cluster. Argo CD pulls from `athena-gitops` and needs no credentials from CI. The production Argo CD manages only the production cluster, and the beta Argo CD manages only the beta cluster.

- `production`: automated sync, `selfHeal: true`, `prune: true`. A merged MR is the approval.
- `beta`: automated sync, same settings. Beta syncs whatever its values file says, and that file changes only by a merged `deploy:beta` MR.
- Migrations run as an Argo CD `PreSync` hook job, before the Deployments change. The migration job runs once per sync.

Rollback is a Git revert of the environment values commit. Argo CD syncs the previous tag back. The old images are still in the registry, so nothing is rebuilt. See `docs/rollback.md`.

### Credentials and permissions

| Identity | Can do | Cannot do |
|---|---|---|
| App repo CI (`backend`, `web`) | Build, push to the registry, read the GitOps repo | Write to the GitOps repo, reach the cluster |
| GitOps CI | Open and update MRs in `athena-gitops` through a project access token with `write_repository` | Push images, reach the cluster |
| Argo CD | Read `athena-gitops`, apply manifests to its namespaces | Change Git |
| Humans | Approve MRs, create release tags | Apply manifests by hand in production |

- Each GitLab project has its own CI variables. Registry credentials and the GitOps token are masked and protected.
- The `production` GitLab environment has a deployment tier of `production` and requires an approval for the job that changes the production values.
- Argo CD gets a namespace-scoped `AppProject` for beta and production, not cluster-admin.

### Scanning and supply chain

| Control | Where | Failure rule |
|---|---|---|
| Secret scanning (`gitleaks`) | Every MR and `main` push, all repos | Any finding fails the pipeline |
| Dependency scanning (`govulncheck`, `npm audit`) | Every MR and `main` push | High or critical with a fix available fails |
| Image scanning (Trivy) | After each image build | High or critical with a fix available fails |
| SBOM (Syft) | At image build, attached to the image | Required, no fail rule |
| Image signing (cosign) | After push to registry | Admission policy in the cluster verifies the signature (not built) |

### What changes from Part 1

| Area | Today | Target |
|---|---|---|
| Repositories | One repo | Four repos |
| CI | GitHub Actions | GitLab CI per repo |
| Registry | Docker Hub | GitLab Container Registry |
| Release images | Rebuilt from the tag | Promoted by digest from `main` |
| Production deploy | Operator runs `make app up` | Merged MR in `athena-gitops`, synced by Argo CD |
| Beta | Does not exist | Manual `deploy:beta` with any tag |
| Cluster credentials | Admin kubeconfig on the controller | Argo CD in-cluster, namespace-scoped |

### Decisions

- GitLab: the company's hosted GitLab is used. The registry is the one that comes with it.
- Argo CD: designed to run on each cluster. Not installed yet.
- Beta: a separate cluster from production, with its own Argo CD.
- `main` does not deploy to beta automatically. Deploys are driven by tags and merged MRs only.

### Gaps against production (Part 2)

- Nothing in Part 2 is built. The four repositories, GitLab pipelines, GitOps repo, and Argo CD do not exist yet.
- The backend still has no test files, so the MR and `main` test jobs pass without testing behavior.
- Beta data isolation depends on a separate PostgreSQL, which is not provisioned.
