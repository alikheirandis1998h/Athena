# kubeadm_control_plane

Creates the cluster on the first control-plane node and joins the other two to it. Runs on the `control_plane` group.

## What it does

- Renders the kubeadm configuration. Images come from `{{ registry_url }}`, and etcd is pinned to `v3.6.6`.
- Binds kube-apiserver to each node's own address, because HAProxy owns the VIP port 6443.
- Runs `kubeadm init` on `control_plane[0]` once. Skips it if `/etc/kubernetes/admin.conf` exists.
- Skips the kube-proxy addon, because Cilium replaces kube-proxy.
- When a control-plane node still needs to join, it creates a short-lived join token (`kubeadm_token_ttl`) and uploads a certificate key. The key is removed once the joins finish.
- Joins the remaining control-plane nodes one at a time, through the VIP.

## Limits

- Changes to the kubeadm configuration are not applied to a running cluster. Make those changes by hand, or rebuild the cluster.
- If an `init` or `join` fails part way, the role won't recover it. Reset the affected node with `sudo kubeadm reset -f`, then rerun `make k8s up`.
- The first control-plane node is the first host in the inventory group. Reordering that group changes which node initializes the cluster.

## Verification

- `kubectl get nodes` lists all three control-plane nodes as Ready.
- A second `make k8s up` changes nothing.
