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

all: infra-dev db-to-code

# ── Code ──────────────────────────────────────────────────────────────────────

db-to-code:
	@echo "--- 🐘 1. Extraction du schéma PostgreSQL ---"
	pg_dump -s -x -O -d $(DB_URL) -T "databasechangelog*" > $(BACK_DIR)/$(SCHEMA_FILE_POSTGRES)

	@echo "--- Extraction du schéma Mariadb ---"
	docker exec mon-mysql mariadb-dump -u root -proot --no-data cyber_notes_v2 > $(BACK_DIR)/$(SCHEMA_FILE_MARIADB)

	@echo "--- 🧹 2. Nettoyage du fichier SQL ---"
	sed -i '/restrict/d' $(BACK_DIR)/$(SCHEMA_FILE_POSTGRES)
	sed -i '/unrestrict/d' $(BACK_DIR)/$(SCHEMA_FILE_POSTGRES)
	sed -i '/^--/d' $(BACK_DIR)/$(SCHEMA_FILE_POSTGRES)

	@echo "--- 🏗️ 3. Lancement de sqlc generate ---"
	cd $(BACK_DIR)/admin && sqlc generate
	cd $(BACK_DIR)/common && sqlc generate
	cd $(BACK_DIR)/student && sqlc generate

# ── Nettoyage ─────────────────────────────────────────────────────────────────

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose down
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs
