include vm/vm.env
export

.PHONY: host-setup vm k8s app up down status destroy deploy

VM_ACTIONS := up down status destroy
APP_ACTIONS := deploy

# The commit that CI built the images from. Images are tagged api-<sha> and web-<sha>.
APP_SHA ?= $(shell git rev-parse HEAD)

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

app:
	@case "$(word 2,$(MAKECMDGOALS))" in \
		deploy) cd ansible && ansible-playbook playbooks/app-deploy.yaml --ask-become-pass -e app_release_sha=$(APP_SHA) ;; \
		*) echo "usage: make app deploy [APP_SHA=<commit sha>]" >&2; exit 2 ;; \
	esac

$(VM_ACTIONS):
	@$(if $(filter vm k8s,$(MAKECMDGOALS)),,echo "usage: make vm|k8s $@" >&2; exit 2)

$(APP_ACTIONS):
	@$(if $(filter app,$(MAKECMDGOALS)),,echo "usage: make app deploy" >&2; exit 2)
