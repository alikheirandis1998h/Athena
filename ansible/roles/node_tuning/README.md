# node_tuning

Prepares Ubuntu 24.04 guests for kubeadm, containerd, and Longhorn. Idempotent: re-running changes nothing on a converged node.

Kubelet reservations, eviction thresholds, and graceful shutdown are kubelet settings. They are set in the kubeadm configuration (`kubeadm-config.yaml.j2`) for new clusters, and merged into `/var/lib/kubelet/config.yaml` on nodes that already joined (`kubernetes_base`). The values are in `ansible/inventories/group_vars/kubernetes.yaml`. The systemd-logind setting is written by `kubernetes_base` as well.

## Settings

Each row gives the problem, the reason for the value, the downside, how production would choose it, and how it is verified.

| Setting | Value | Problem | Why this value | Downside | Production | Verify |
|---|---|---|---|---|---|---|
| Swap disabled, swap file removed | off | kubelet refuses to start with swap enabled by default; swap hides memory pressure from the scheduler | Kubelet requires it | Less headroom before OOM | Keep swap off for Kubernetes nodes | `swapon --show` empty; no `/swap.img` |
| `overlay`, `br_netfilter`, `nf_conntrack` | loaded, persisted | containerd needs overlayfs; bridged pod traffic must reach iptables; the conntrack sysctl only exists once the module is loaded | Required by containerd, kube-proxy/CNI, and the conntrack setting below | None | Same | `lsmod` shows all three; `/etc/modules-load.d/athena.conf` |
| `iscsi_tcp` | loaded, persisted | Longhorn attaches volumes over iSCSI | Required by Longhorn | Module is loaded even on nodes without volumes | Same, with `open-iscsi` installed | `lsmod` shows `iscsi_tcp` |
| `net.ipv4.ip_forward` | 1 | Pods need to route traffic off their node | Required by the CNI | Allows forwarding if misconfigured | Same | `sysctl net.ipv4.ip_forward` |
| `net.bridge.bridge-nf-call-iptables` / `-ip6tables` | 1 | Service and NetworkPolicy rules must see bridged pod traffic | Required by kube-proxy and most CNIs | None beyond the module | Same | `sysctl` values |
| `net.netfilter.nf_conntrack_max` | 262144 | Ingress and service traffic fills the connection-tracking table, and new connections fail with `nf_conntrack: table full` | Larger than the kernel default for a 3 GiB node; each entry is small, so the table stays modest in memory | Higher memory use under load | Size from measured connection counts on production nodes, not a fixed formula | `sysctl net.netfilter.nf_conntrack_max`; compare `conntrack -C` with the max |
| `net.core.somaxconn` | 4096 | Ingress accept queue overflows under bursts | Enough headroom for bursty HTTP without a large memory cost | Longer queues hide overload | Tune from load tests | `sysctl net.core.somaxconn` |
| `fs.inotify.max_user_watches` / `max_user_instances` | 524288 / 512 | kubelet, containerd, and log agents exhaust inotify watches on nodes with many pods | Well above the defaults, leaving room for many pods per node | Kernel memory per watch | Scale with pod count | `sysctl` values |
| `fs.file-max` | 2097152 | Many sockets and files per node | Well above the expected per-node load | Kernel memory, negligible | Scale with pod density | `sysctl fs.file-max` |
| `DefaultLimitNOFILE` (systemd) | 1048576 | containerd and kubelet hit the default per-process file-descriptor limit | Large enough for many containers and connections per node | Kernel memory, negligible | Same | `systemctl show -p DefaultLimitNOFILE` |
| Journal cap | 500M | Logs fill the disk on small VM disks | Caps disk use and leaves room for images | Older logs roll off sooner | Ship logs off-node and cap locally | `journalctl --disk-usage` |
| chrony | enabled, `systemd-timesyncd` disabled | Certificate validation and etcd leases fail on clock drift | chrony tracks drift better than timesyncd | One more service | Use NTP sources reachable from the production network | `chronyc tracking`; `timedatectl` shows NTP synchronized |
| `systemReserved` (kubelet) | cpu 100m, memory 256Mi | sshd, journald, chrony, and the rest of the OS compete with pods, and the scheduler would hand out all 3 GiB | Covers these daemons on a 3 GiB node. Small enough to leave room for the workloads here | Less memory is schedulable. The reservation is only subtracted from what the scheduler sees, not enforced (`enforceNodeAllocatable` stays at its default, `pods`) | Measure the memory of `system.slice` on production nodes, then reserve that plus a margin | `kubectl get node <n> -o jsonpath='{.status.allocatable}'` is below capacity by at least the reserved amount |
| `kubeReserved` (kubelet) | cpu 100m, memory 256Mi | kubelet and containerd use memory that pods cannot account for | Covers kubelet and containerd at the pod counts used here (about 20 pods per node) | Same as `systemReserved` | Measure the kubelet and containerd cgroups under load, then reserve that plus a margin | Same check as `systemReserved` |
| `evictionHard` (kubelet) | `memory.available` 100Mi, `nodefs.available` 10%, `nodefs.inodesFree` 5%, `imagefs.available` 15% | Without eviction, the node runs out of memory or disk first and the kernel OOM killer or a full disk picks the victim, and node services fail | The kubelet defaults. The threshold is the last resort, so the kubelet evicts pods before the kernel acts | A full node evicts pods, which can include a PostgreSQL pod on its Longhorn volume | Size from the disk and memory of the production node, and keep the kubelet defaults as the floor | `kubectl describe node` shows the thresholds under Conditions; `MemoryPressure` and `DiskPressure` are False |
| `evictionSoft` and `evictionSoftGracePeriod` (kubelet) | `memory.available` 300Mi, held for 90s | A sudden drop to the hard threshold gives no time to shed load, so the kubelet evicts at once | 300Mi is about 10% of a 3 GiB node. 90s lets a short spike pass without an eviction | Pods are evicted earlier when memory is tight but not yet critical | Set the soft threshold as a share of production node memory. Keep the grace period above the normal peak duration | `kubectl describe node` lists the soft thresholds; the kubelet config file has `evictionSoft` |
| `shutdownGracePeriod` and `shutdownGracePeriodCriticalPods` (kubelet) | 30s total, 10s for critical pods | When a node reboots or powers off, the kubelet kills pods at once, so in-flight API requests fail | The API stops within 10s (`app/backend/cmd/api/main.go`). Regular pods get 20s, and system pods get 10s | A node shutdown takes up to 30s longer | Use the longest application drain time plus a margin. Stay below the VM or cloud stop timeout | During a poweroff, the kubelet journal shows the shutdown manager. Not yet run on this cluster |
| `InhibitDelayMaxSec` (systemd-logind) | 60s | systemd caps an inhibitor delay at 5s by default, so the kubelet's `shutdownGracePeriod` would be cut to 5s | Must be above `shutdownGracePeriod` (30s), with room for the kubelet's own delay | Any service that takes an inhibitor lock can delay a shutdown by up to 60s | Same value as the longest shutdown grace period, plus margin | `systemd-analyze cat-config systemd/logind.conf` shows the drop-in. `Restart logind` runs when the file changes |

## Verification

Run `make k8s up` twice. The second run must report no changes. Then check the settings in the table above on any node.

## Limits of the local setup

- The VMs share one host, so kernel tuning here does not match production hardware.
- The conntrack table size is set for 3 GiB nodes. Production nodes need it recalculated from their real memory.
- The kubelet settings reach existing nodes through `kubernetes_base`. Nodes that join later get their config from the `kubelet-config` ConfigMap, which `kubeadm init` wrote. That ConfigMap keeps the old values on a cluster built before this change. Update it with `kubeadm upgrade` or by hand before adding nodes.
