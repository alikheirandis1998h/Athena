# kubeadm_worker

Joins the three worker nodes to the cluster through the HAProxy VIP. Runs on the `workers` group.

## What it does

- Checks whether each worker has already joined (`/etc/kubernetes/kubelet.conf` exists).
- If any worker still needs to join, creates a one-hour join token on the first control-plane node and the cluster CA hash for verification.
- Renders a join configuration with the token and hash, runs `kubeadm join`, then deletes the configuration file.

## Limits

- The join token TTL is fixed at one hour in the role. The control-plane role uses the `kubeadm_token_ttl` variable, so the two don't share one setting.
- If a join fails part way, reset the worker with `sudo kubeadm reset -f`, then rerun `make k8s up`.

## Verification

- `kubectl get nodes` lists all three workers as Ready.
- A second `make k8s up` changes nothing.
