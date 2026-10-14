# longhorn

Installs Longhorn block storage. The role has two parts:

- `tasks/workers.yml`: installs and starts the iSCSI initiator (`open-iscsi`, `iscsid`) on the workers. Longhorn attaches volumes over iSCSI. The role runs this part from a play on the `workers` group.
- `tasks/main.yml`: installs Longhorn on the first control-plane node through Helm.

## What it does

- Downloads the pinned Longhorn 1.13.0 chart bundle from GitHub. The download is checked against `longhorn_charts_sha256`.
- Renders the values. Every image comes from `{{ registry_url }}` through `global.imageRegistry`.
- Sets the default replica count to `longhorn_replica_count` (3, one per worker).
- Installs or upgrades the release when the values change, or when the release is missing or not `deployed`.
- Waits for the manager DaemonSet and the `longhorn` StorageClass.

## Limits

- Backups are disabled. The Silo backup target isn't ported yet, so there is no backup target.
- Longhorn only runs on the workers. The control-plane nodes don't store volume data.

## Verification

- `kubectl -n longhorn-system get pods` shows all pods Running or Completed.
- `kubectl get storageclass longhorn` exists.
- A second `make k8s up` changes nothing.
