# kubernetes_base

Installs the Kubernetes node software on all six cluster nodes: containerd, kubelet, kubeadm, and kubectl, at the versions pinned in `group_vars/kubernetes.yaml`.

## What it does

- Adds the pkgs.k8s.io apt repository for the pinned minor version. The signing key is checked against `kubernetes_apt_key_sha256`.
- Installs the pinned containerd and Kubernetes packages, then holds kubelet, kubeadm, and kubectl so unattended upgrades don't move them.
- Configures containerd to use the systemd cgroup driver, and points the sandbox (pause) image at `{{ registry_url }}`.
- Points crictl at containerd and sets the kubelet node IP to the node's cluster address.
- Enables kubelet. It starts for real during `kubeadm init` or `join`.

## Limits

- Packages come from the public Kubernetes repository. Only the container images use `registry_url`.
- The containerd config is generated once, so later changes to the defaults need a manual edit.

## Verification

- `kubeadm version`, `kubelet --version`, and `containerd --version` show the pinned versions.
- `apt-mark showhold` lists kubelet, kubeadm, and kubectl.
