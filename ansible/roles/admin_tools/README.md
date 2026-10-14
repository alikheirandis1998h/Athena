# admin_tools

Sets up the admin tools on the first control-plane node (`cp-1`): kubectl and k9s. Run them there, not on the controller host.

## What it does

- Installs the pinned k9s 0.51.0 release into `/usr/local/bin`. The download is checked against `k9s_sha256`.
- Copies `/etc/kubernetes/admin.conf` to `~/.kube/config` for the login user (`vagrant`) and to `/root/.kube/config` for root. Both are mode 0600, so `kubectl` and `k9s` work without `--kubeconfig`, with or without `sudo`.

kubectl comes from the `kubernetes_base` role. It is already installed on every node.

## Limits

- The admin kubeconfig gives full cluster access. It stays on cp-1 and is never copied to the controller host.
- The copy is refreshed on every run. If you rotate the cluster certificates, rerun `make k8s up` to update it.

## Verification

- On cp-1, `kubectl get nodes` works with no flags.
- `k9s version` prints v0.51.0.
- `sudo kubectl get nodes` also works with no flags.
