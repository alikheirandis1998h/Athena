# host_setup

**Implemented:** controller-host preparation for Vagrant and libvirt, run against `localhost` with `connection: local`. It replaces `scripts/host-setup.sh`.

**Designed but not implemented:** none.

**Out of scope:** the VMs themselves (`make up`) and any remote-target configuration.

## What the role does

- Installs the APT packages listed in `host_setup_apt_packages`.
- Installs the pinned Vagrant `.deb` when the installed version differs. The download is checked against `host_setup_vagrant_sha256`.
- Installs the pinned `vagrant-libvirt` plugin when it is missing.
- Starts and enables `libvirtd`, and adds the user to the `libvirt` group.
- Writes the `libvirtd` drop-in that grants the user an ACL on the libvirt socket. It reloads systemd and restarts `libvirtd` only when the drop-in changes. The ACL itself is set with `ansible.posix.acl`.
- Warns when `/dev/kvm` is not writable. It does not fail, because KVM is optional.

## Inputs

- Pins are role defaults in `roles/host_setup/defaults/main.yml`. Override them with `-e` or group/host vars if needed.
- The user is the account that runs `ansible-playbook`. Run it with `--ask-become-pass` so sudo can prompt.

## Differences from the shell script

- APT metadata is refreshed at most once an hour (`cache_valid_time`). The script refreshed it only when a package was missing.
- Vagrant and the plugin are checked and installed as the user. Only the package and service changes use sudo.

## Verification

- `ansible-playbook playbooks/host-setup.yaml --ask-become-pass` on a prepared host reports `changed=0` on a second run.
- `vagrant --version` prints the pinned version, and `vagrant plugin list` shows `vagrant-libvirt (0.12.2)`.
- `getfacl /var/run/libvirt/libvirt-sock` shows `user:<you>:rw-`.
