# helm

Installs the pinned Helm 3 client on the first control-plane node. The cilium, longhorn, and ingress_nginx roles use it to install charts.

## What it does

- Downloads the Helm release from get.helm.sh. The download is checked against `helm_sha256`.
- Unpacks it into `/opt/helm/<version>` and links the `helm` binary into `/usr/local/bin`.

## Verification

- `helm version` prints the pinned version.
- A second run changes nothing, because the download and unpack steps are skipped when they are current.
