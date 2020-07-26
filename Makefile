ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PLAYBOOK?=router.yml
UID:=$(shell id -u)

.PHONY: playbook
playbook:
		docker run -w /project -e HOME=/project --rm -it \
			-v $(ROOT_DIR)/ansible:/project \
			-v $(ROOT_DIR)/known_hosts:/known_hosts \
			-v $(SSH_AUTH_SOCK):/ssh-agent \
			-e SSH_AUTH_SOCK=/ssh-agent \
			-u $(UID) \
			ansible/ansible-runner ansible-playbook $(PLAYBOOK) -i inventory.yml

# For debugging: run make shell
.PHONY: shell
shell:
		docker run -w /project -e HOME=/project --rm -it \
			-v $(ROOT_DIR)/ansible:/project \
			-v $(ROOT_DIR)/known_hosts:/known_hosts \
			-v $(SSH_AUTH_SOCK):/ssh-agent \
			-e SSH_AUTH_SOCK=/ssh-agent \
			-u $(UID) \
			ansible/ansible-runner bash