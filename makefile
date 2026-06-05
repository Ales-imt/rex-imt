.DEFAULT_GOAL := all

# --- Variables communes ---
SECRETS_FILE_DEV=.vscode/secrets-dev.env
SECRETS_FILE_PROD=.vscode/secrets-prod.env

_get_dev=$(shell grep '^$(1)=' $(SECRETS_FILE_DEV) | cut -d= -f2)
POSTGRES_USER=$(call _get_dev,POSTGRES_USER)
POSTGRES_PASSWORD=$(call _get_dev,POSTGRES_PASSWORD)
POSTGRES_HOST=$(call _get_dev,POSTGRES_HOST)
POSTGRES_PORT=$(call _get_dev,POSTGRES_PORT)
POSTGRES_DB=$(call _get_dev,POSTGRES_DB)
DB_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)

SCHEMA_FILE_POSTGRES=schema.sql
SCHEMA_FILE_MARIADB=schema_maria_db.sql
BACK_DIR=./backend
INFRA_DIR=./infras
DOCKER_DIR=./infras/container
RUN_DIR=./infras/run
ANSIBLE_DIR=./infras/ansible
ADMIN_CONTAINER=rex-admin
ELEVE_CONTAINER=rex-eleve

include makefile.dev
include makefile.prod

.PHONY: all db-to-code clean

all:
	@echo ""
	@echo "Usage : make dev   — déploiement local"
	@echo "        make prod  — déploiement production"
	@echo ""


# ── Nettoyage ─────────────────────────────────────────────────────────────────

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose down
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs
