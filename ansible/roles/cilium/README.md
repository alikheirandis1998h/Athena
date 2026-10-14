# cilium

Installs Cilium as the pod network and replaces kube-proxy. Runs on the first control-plane node through Helm.

## What it does

- Renders the Cilium values. The images come from `{{ registry_url }}`, and the chart comes from `oci://quay.io/cilium/charts/cilium`, pinned by digest.
- Sets `kubeProxyReplacement`, with the API server reached through the HAProxy VIP.
- Uses VXLAN tunnels, with cluster-pool IPAM in `kubernetes_pod_subnet`.
- Installs or upgrades the release when the values change, or when the release is missing or not `deployed`.
- Waits for the Cilium DaemonSet and for every node to become Ready.

## Limits

- Images are pinned by tag, not digest (`useDigest: false`). The chart is pinned by digest.
- The device is `eth1`. Other network layouts need `cilium_device` changed.

## Verification

- `kubectl -n kube-system get ds cilium` shows all agents ready.
- `kubectl get nodes` shows every node Ready.
- A second `make k8s up` changes nothing.
