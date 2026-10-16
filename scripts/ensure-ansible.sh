#!/usr/bin/env bash
# Checks that ansible-playbook is on the controller and installs Ansible from the system package manager if it is missing.
set -euo pipefail

if command -v ansible-playbook >/dev/null 2>&1; then
  echo "Ansible found: $(ansible --version | head -n 1)"
  exit 0
fi

echo "Ansible is not installed. Installing it..."

run_root() {
  if [ -n "${ATHENA_BECOME_PASSWORD:-}" ]; then
    printf '%s\n' "$ATHENA_BECOME_PASSWORD" | sudo -S -p '' "$@"
  else
    sudo "$@"
  fi
}

if command -v apt-get >/dev/null 2>&1; then
  run_root apt-get update
  run_root apt-get install -y ansible
elif command -v dnf >/dev/null 2>&1; then
  run_root dnf install -y ansible
else
  echo "No supported package manager found (apt-get or dnf). Install Ansible manually and rerun make up." >&2
  exit 1
fi

if ! command -v ansible-playbook >/dev/null 2>&1; then
  echo "Ansible installation finished, but ansible-playbook is still not on PATH." >&2
  exit 1
fi

echo "Ansible installed: $(ansible --version | head -n 1)"
