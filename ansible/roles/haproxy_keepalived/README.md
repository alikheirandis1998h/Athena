# haproxy_keepalived

Gives kubeadm one stable control-plane endpoint, `192.168.56.10:6443`, on the three control-plane nodes.

**Implemented:** HAProxy forwards TCP 6443 on the VIP to `athena-cp-1..3:6443`. keepalived moves the VIP between the three nodes with VRRP unicast.

**Designed but not implemented:** TLS passthrough checks against the API server, and a separate load-balancer VM (the design keeps HAProxy on the control-plane nodes for local simplicity).

## Design

- **HAProxy binds only the VIP**, not `0.0.0.0`. kube-apiserver must bind the node IP (kubeadm configuration, next phase), so the two never collide on port 6443.
- **`ip_nonlocal_bind=1`** lets HAProxy start on nodes that don't hold the VIP.
- **Priorities 150, 140, 130** follow the inventory order. `nopreempt` keeps the VIP where it is after a failover, so it doesn't move back and forth.
- **Health check:** keepalived lowers a node's priority by 20 when the local HAProxy process is gone. It does not check the backends. Before kubeadm runs, every backend is down, and the VIP must still stay put.
- **VRRP secret:** generated once on `athena-cp-1` into `/etc/keepalived/auth.env` (mode 0600). Other nodes read it through Ansible at run time and keep it only in the rendered, root-only `keepalived.conf`. Nothing is in Git.

## Limits

- VRRP `PASS` authentication uses only the first 8 characters of the secret. It guards against accidental joins on the private network. It is not a strong security control.
- Unicast VRRP needs every peer listed. A new control-plane node means editing the group, which the template picks up automatically.
- Single subnet only (`192.168.56.0/24`). Production would use a dedicated VIP subnet or cloud load balancer.

## Verification

- `make k8s up` twice: the second run must report `changed=0` on the control-plane nodes.
- Exactly one node holds `192.168.56.10` on `eth1`, and `haproxy` and `keepalived` are both `active`.
- Failover: stopping keepalived on the holder moves the VIP to the next priority, and the VIP stays there when the first node comes back. Not yet run.
