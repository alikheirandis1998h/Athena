include vm/vm.env
export

.PHONY: host-setup vm k8s up down status destroy

VM_ACTIONS := up down status destroy

host-setup:
	cd ansible && ansible-playbook playbooks/host-setup.yaml --ask-become-pass

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

$(VM_ACTIONS):
	@$(if $(filter vm k8s,$(MAKECMDGOALS)),,echo "usage: make vm|k8s $@" >&2; exit 2)
