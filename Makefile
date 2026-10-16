include vm/vm.env
export

SHELL := /bin/bash

.PHONY: host-setup vm k8s app test failure up down status destroy

VM_ACTIONS := down status destroy
FAILURE_ACTIONS := pod node

# Release to deploy, as a vX.Y.Z-release tag. Defaults to the latest such tag.
APP_RELEASE ?= $(shell git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*-release' 2>/dev/null)

# Ask once for the controller's sudo password and export it for Ansible (ansible/inventories/host_vars/localhost.yaml).
# Skipped when it is already set, so make up prompts once for all its steps.
ASK_BECOME = if [ "$$(id -u)" -eq 0 ]; then echo "Run make as your normal user, not with sudo. It asks for the sudo password itself." >&2; exit 1; fi; if [ -z "$$ATHENA_BECOME_PASSWORD" ]; then read -r -s -p "sudo password for $$USER: " ATHENA_BECOME_PASSWORD && echo && export ATHENA_BECOME_PASSWORD; fi

host-setup:
	@$(ASK_BECOME); cd ansible && ansible-playbook playbooks/host-setup.yaml

up:
	@$(if $(filter vm k8s app,$(MAKECMDGOALS)),,$(ASK_BECOME); bash scripts/ensure-ansible.sh && $(MAKE) host-setup && $(MAKE) vm up && $(MAKE) k8s up && $(MAKE) app up)

vm:
	@case "$(word 2,$(MAKECMDGOALS))" in \
		up) cd vm && vagrant up --provider=libvirt ;; \
		down) cd vm && vagrant halt ;; \
		status) cd vm && vagrant status ;; \
		destroy) cd vm && vagrant destroy -f ;; \
		*) echo "usage: make vm {up|down|status|destroy}" >&2; exit 2 ;; \
	esac

k8s:
	@case "$(word 2,$(MAKECMDGOALS))" in \
		up) cd ansible && ansible-playbook playbooks/k8s-setup.yaml ;; \
		status) cd ansible && ansible-playbook playbooks/k8s-status.yaml ;; \
		*) echo "usage: make k8s {up|status}" >&2; exit 2 ;; \
	esac

app:
	@case "$(word 2,$(MAKECMDGOALS))" in \
		up) $(ASK_BECOME); cd ansible && ansible-playbook playbooks/app-deploy.yaml -e app_release_sha=$$(git rev-parse --verify "$(APP_RELEASE)^{commit}") -e app_release_tag=$(APP_RELEASE) ;; \
		*) echo "usage: make app up [APP_RELEASE=vX.Y.Z]" >&2; exit 2 ;; \
	esac

test:
	cd ansible && ansible-playbook playbooks/acceptance-test.yaml

failure:
	@case "$(word 2,$(MAKECMDGOALS))" in \
		pod) cd ansible && ansible-playbook playbooks/failure-pod.yaml ;; \
		node) cd ansible && ansible-playbook playbooks/failure-node.yaml ;; \
		"") cd ansible && ansible-playbook playbooks/failure-pod.yaml && ansible-playbook playbooks/failure-node.yaml ;; \
		*) echo "usage: make failure [pod|node]" >&2; exit 2 ;; \
	esac

$(FAILURE_ACTIONS):
	@$(if $(filter failure,$(MAKECMDGOALS)),,echo "usage: make failure [pod|node]" >&2; exit 2)

$(VM_ACTIONS):
	@$(if $(filter vm k8s,$(MAKECMDGOALS)),,echo "usage: make vm|k8s $@" >&2; exit 2)
