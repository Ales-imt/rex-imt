.DEFAULT_GOAL := all

# --- Variables communes ---
SECRETS_FILE_LOCAL=.vscode/secrets-local.env
SECRETS_FILE_PROD=.vscode/secrets-prod.env

SCHEMA_FILE_POSTGRES=schema.sql
SCHEMA_FILE_MARIADB=schema_maria_db.sql
BACK_DIR=./backend
INFRA_DIR=./infras
DOCKER_DIR=./infras/container
RUN_DIR=./infras/run
ANSIBLE_DIR=./infras/ansible
ADMIN_CONTAINER=rex-admin
ELEVE_CONTAINER=rex-eleve

include makefile.local
include makefile.prod

.PHONY: all db-to-code clean

all:
	@echo ""
	@echo "Usage : make local — déploiement local"
	@echo "        make prod  — déploiement production"
	@echo ""


# ── Nettoyage ─────────────────────────────────────────────────────────────────

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose --env-file ../../$(SECRETS_FILE_LOCAL) down
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs
