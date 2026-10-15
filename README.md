# newAthena

A local Kubernetes platform on libvirt VMs, built and managed with Ansible. The VMs, the host preparation, and the cluster all come from this repository, so the environment can be recreated from version control.

## Layout

| Path | Purpose |
|---|---|
| `Makefile` | Entry points: `make host-setup`, `make vm ...`, `make k8s ...` |
| `vm/` | Vagrantfile and box pins. Builds one VM per host in the inventory |
| `ansible/inventories/hosts.yaml` | The seven VMs and their groups |
| `ansible/inventories/group_vars/` | Shared pins and settings (versions, digests, registry, VIP) |
| `ansible/playbooks/host-setup.yaml` | Prepares the controller host (runs on localhost) |
| `ansible/playbooks/k8s-setup.yaml` | Brings up the whole cluster, in order |
| `ansible/playbooks/k8s-status.yaml` | Read-only validation of the nodes and the cluster |
| `ansible/roles/` | One role per component. Each role has a README |

## Prerequisites

- Ubuntu 24.04 controller with KVM (`/dev/kvm`) and libvirt.
- Ansible on the controller.
- Passwordless or prompted sudo for your user.
- Network access to `docker.novin.cloud` (container images), `pkgs.k8s.io`, `quay.io`, `ghcr.io`, and `github.com`.
- If `/usr/bin/sudo` is sudo-rs, export `ANSIBLE_BECOME_EXE=/usr/bin/sudo.ws` before `make host-setup`. See `ansible/roles/host_setup/README.md`.

## Usage

Run the steps in this order:

```
make host-setup     # once per controller: Vagrant, libvirt, and the libvirt ACL
make vm up          # create and start the seven VMs
make k8s up         # configure the VMs and bring up the cluster
make k8s status     # validate the nodes and the cluster; fails if any check fails
make app deploy     # deploy the sample app at the latest vX.Y.Z-release tag
make test           # the full acceptance test; fails and names the failed checks
make failure        # delete an API pod and measure recovery; see docs/failure-exercise.md
```

Other commands:

```
make vm down        # halt the VMs and keep their disks
make vm status      # show VM state
make vm destroy     # delete the VMs. Never run automatically
```

To inspect the cluster, log in to cp-1 and run `kubectl` or `k9s` there. Both use the admin kubeconfig already in place, so no flag is needed:

```
ssh -i vm/.vagrant/machines/athena-cp-1/libvirt/private_key vagrant@192.168.56.11
kubectl get nodes
k9s
```

Every `make k8s up` step is safe to repeat. A second run on a healthy cluster changes nothing.

## Topology

| Node | Address | Role |
|---|---|---|
| `athena-cp-1..3` | 192.168.56.11–13 | Control plane (etcd, API server), HAProxy, keepalived |
| `athena-worker-1..3` | 192.168.56.21–23 | Workloads, Longhorn volumes |
| `athena-minio` | 192.168.56.31 | Created but not configured yet (Silo is not ported) |
| API VIP | 192.168.56.10:6443 | Control-plane endpoint, held by keepalived |

The `make k8s up` play order is `node_tuning`, `haproxy_keepalived`, `kubernetes_base`, `kubeadm_control_plane`, `kubeadm_worker`, the Longhorn iSCSI tasks on the workers, then `helm`, `cilium`, `longhorn`, and `ingress_nginx` on the first control-plane node.

## Versions and registry

- Every version is a variable in `ansible/inventories/group_vars/` or a role's `defaults/`. Nothing is pinned in a separate file.
- Container images are pulled from `registry_url` (`docker.novin.cloud`). It is set in `group_vars/kubernetes.yaml`. Change it there to move the whole cluster to another registry.
- Helm charts keep their upstream sources. Cilium and F5 charts are pinned by digest. The Longhorn chart is pinned by checksum.

## Local simplifications and production design

The assignment requires these to be named. Each one is a deliberate local choice, with the production design alongside.

| Local choice | Production design |
|---|---|
| All VMs on one libvirt host | Separate hosts or cloud instances across failure domains |
| HAProxy and keepalived on the control-plane nodes | Dedicated load-balancer tier, or a cloud load balancer |
| Unicast VRRP on one `/24` subnet, with an 8-character VRRP secret | Dedicated VIP subnet and a stronger authentication setup |
| Ingress exposed as NodePorts 30080/30443 on the workers | External load balancer in front of the NodePorts or a LoadBalancer service |
| Longhorn with 3 replicas, backups disabled | Longhorn backups to an S3-compatible target, with a tested restore procedure |
| `host_key_checking = False` for Ansible SSH | Host keys verified against a known-hosts file |
| Images from one registry, using tags for Cilium images | Digest-pinned images and an image signing and scanning process |
| Helm installs run from Ansible on the first control-plane node | GitOps, with one owner for each release |
| Kernel tuning sized for 3 GiB nodes | Sized from measured load on production hardware |
| Secrets generated on the cluster and kept in root-only files | A secret store, such as Vault or an external secrets operator |

## Current state and known gaps

Done: VM provisioning, host preparation, node tuning, the highly available control plane, the worker join, Cilium, Longhorn storage, the F5 NGINX ingress, the `k8s status` validation, the sample app with its tag-based deploy, `make test`, and the API pod failure exercise (`make failure`, see `docs/failure-exercise.md`).

Not done yet (see `TaskDefinition.md`):

- The app runs two replicas each for the API and web, with a PodDisruptionBudget (`minAvailable: 1`). Not yet deployed or re-run with `make failure`. Both replicas can still land on the same worker, because there is no anti-affinity.
- The app does not run as non-root and has no `securityContext`.
- The observability layer (metrics, dashboards, or alerting).
- Rollback and observability design write-ups.
- Longhorn backups, which depend on porting the Silo role.
- The Patroni and athena_app roles from the Athena project.

## Troubleshooting

- `make k8s status` names the failed check. Run the matching `kubectl` command from the role README to see details.
- Pods stuck in `ImagePullBackOff`: check the pull error with `kubectl describe pod`. Image paths must start with `{{ registry_url }}`.
- A failed `kubeadm init` or `join`: reset the affected node with `sudo kubeadm reset -f`, then rerun `make k8s up`. The roles don't do this automatically.
