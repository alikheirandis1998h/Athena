# node_tuning

Prepares Ubuntu 24.04 guests for kubeadm, containerd, and Longhorn. Idempotent: re-running changes nothing on a converged node.

Kubelet reservations, eviction thresholds, and graceful shutdown belong in the kubeadm configuration, so they are not in this role. They are added with the kubeadm bootstrap.

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

## Verification

Run `make k8s up` twice. The second run must report no changes. Then check the settings in the table above on any node.

## Limits of the local setup

- The VMs share one host, so kernel tuning here does not match production hardware.
- The conntrack table size is set for 3 GiB nodes. Production nodes need it recalculated from their real memory.
