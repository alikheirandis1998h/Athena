# vm

Local Vagrant VMs for the platform, using the libvirt provider.

**Implemented:** seven VMs (`athena-cp-1..3`, `athena-worker-1..3`, `athena-minio`) defined from `ansible/inventories/hosts.yaml`, with the box pinned in `vm.env`.

**Out of scope:** configuring the VMs for Kubernetes. That is `make k8s up`, which runs after the VMs are up.

## Commands

```
make vm up        # create and start the VMs
make vm down      # halt the VMs and keep their disks
make vm status    # show VM state
make vm destroy   # delete the VMs (never run automatically)
```

Run `make host-setup` once first, so Vagrant and the libvirt plugin are installed.

Always run Vagrant through `make vm ...`. The Vagrantfile needs `ATHENA_BOX` and `ATHENA_BOX_VERSION`, which the Makefile exports from `vm.env`. Running `vagrant` directly fails with a missing-variable error.

## Files

- `Vagrantfile`: builds one machine per inventory host in the `vms` group.
- `vm.env`: base box name and version. The Makefile exports them to Vagrant.
- `.vagrant/` (git-ignored): VM state and SSH keys.
