# CI/CD design

Status: partly implemented. The GitHub Actions workflow (`.github/workflows/ci-cd.yml`) implements the build, test, and publish stages. Promotion, approval, scanning, and deploy-permission separation are designed here and not built.

## Pipeline

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

## Stages and what they guarantee

| Stage | Implemented | Guarantee | Gap |
|---|---|---|---|
| Backend checks | `go vet`, `go test` | Code compiles and vet passes | The backend has no test files, so `go test` runs no tests |
| Web checks | `npm ci`, `npm run typecheck` | Dependencies resolve from the lock file; types check | No unit or UI tests |
| Image build | Buildx, build-push, one image per app | The image is built from the commit that passed the checks | Build is not reproducible byte-for-byte |
| Image publish | Only on `main` and `v*-release` | Pull requests cannot push | Publish happens before any cluster test |
| Tagging | `api-<40-hex sha>`, `api-<release tag>`, `api-latest` on `main`; same for `web-` | Each image traces to a commit and, for releases, a tag | `latest` is kept only for `main`; releases are not checked for a signed tag |
| Deploy | `make app up` (Ansible + Helm) | The release tag resolves to a commit, and the chart uses the tag | Deploy is a manual step, run by an operator |
| Verify | `make test` | Pods run the expected images, replicas are ready, the app answers | Runs by hand, after the deploy |

## Promotion without rebuilding

Target design: an image is built once per commit and promoted by digest, never rebuilt for a later stage.

- Today, a release tag causes CI to build the images again from the tagged commit. This is a second build of the same source. It is acceptable for this project, but the release image is not the one that passed the `main` checks byte-for-byte.
- Production design: the `main` build pushes `api-<sha>` and records its digest. Promoting to a release means retagging that digest as `api-<release tag>`, which needs no rebuild. The chart then pins the digest in `values`.
- A release tag must point to a commit that CI already built and tested. `make app up` checks the tag's commit and fails without an image.

## Credentials and permissions

- CI has `contents: read` and only the Docker Hub token as a secret. It cannot write to the repository or to the cluster.
- The deploy uses the controller's kubeconfig (`admin.conf`, cluster-admin), run by an operator. CI has no path to the cluster. This keeps CI off the cluster, but the deploy identity is far broader than needed.
- Production design:
  - A GitHub environment (`production`) with required reviewers. The deploy job runs only in that environment.
  - The deploy identity is a Kubernetes service account bound to the app namespace only, with Helm permissions on that namespace. It is not cluster-admin.
  - Short-lived credentials through OIDC from GitHub to the cluster, so no long-lived kubeconfig is stored in CI.
  - Build identity and deploy identity are separate. The build job cannot deploy, and the deploy job cannot push images.

## Controls to add before production

| Control | Purpose | Where |
|---|---|---|
| Secret scanning (gitleaks) | Catch credentials before merge | CI, on every pull request. `make test` scans tracked files today |
| Dependency scanning (`govulncheck`, `npm audit`) | Known vulnerable dependencies | CI, on every pull request |
| Image scanning (Trivy) | Vulnerable base images and packages | CI, after build, before push. Fail on high or critical with a fix available |
| Image signing (cosign) | Only signed images run | Sign after push; admission policy verifies (not built) |
| SBOM (Syft) | Know what is in each image | Attached to the image at build time |
| Manifest validation (`helm lint`, `kubeconform`, `ansible-lint`) | Catch chart and playbook errors before deploy | CI, on every pull request |
| Environment approval | A person approves each production deploy | GitHub environment `production` |
| Post-deploy verification | Confirm the deploy works | `make test` as the final step of the deploy job |

## Release process

1. Merge to `main`. CI runs the checks and publishes `api-<sha>`, `web-<sha>`, and the `latest` tags.
2. Test on the cluster with `make app up` at the `main` build when needed.
3. Tag the commit `vX.Y.Z-release` and push the tag. CI publishes the release tags.
4. Deploy with `make app up APP_RELEASE=vX.Y.Z-release`, then run `make test`.
5. Record the deploy in the release notes. Keep the rollback target in mind (`docs/rollback.md`).

Tags are immutable. Never move a release tag to another commit. Make a new version instead.

## Gaps against production

- No scanning, signing, or SBOM in the pipeline.
- No environments or approvals.
- Deploy identity is cluster-admin, and the deploy is manual.
- Release images are built again from the tag rather than promoted by digest.
- The backend has no tests, so the checks pass without testing behavior.
