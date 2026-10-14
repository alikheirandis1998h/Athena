# ingress_nginx

Installs the F5 NGINX Ingress Controller, the default ingress class, through Helm on the first control-plane node.

## What it does

- Renders the values. The image comes from `{{ registry_url }}/nginx/nginx-ingress`, pinned to tag 5.6.3 and its digest.
- Installs the chart from `oci://ghcr.io/nginx/charts/nginx-ingress`, pinned by digest.
- Runs `ingress_nginx_replicas` replicas in the `ingress-nginx-f5` namespace.
- Exposes HTTP and HTTPS as NodePorts 30080 and 30443 on every worker. There is no cloud load balancer in this setup.
- Installs or upgrades the release when the values change, or when the release is missing or not `deployed`.
- Waits for the controller rollout.

## Limits

- Traffic reaches the controller through the NodePorts. Put an external load balancer in front of them in production.

## Verification

- `kubectl -n ingress-nginx-f5 get deploy nginx-ingress-controller` shows 2/2 ready.
- `kubectl get ingressclass` shows the controller as the default.
- A second `make k8s up` changes nothing.
