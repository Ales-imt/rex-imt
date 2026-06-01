# --- Variables ---
DB_URL=postgres://postgres:root@localhost:5432/db_rex

SCHEMA_FILE_POSTGRES=schema.sql
SCHEMA_FILE_MARIADB=schema_maria_db.sql
BACK_DIR=./backend
INFRA_DIR=./infras
DOCKER_DIR=./infras/container
RUN_DIR=./infras/run
ADMIN_IMAGE=rex-admin
ADMIN_CONTAINER=rex-admin
ADMIN_CONFIG=$(RUN_DIR)/config-admin.yaml
ELEVE_IMAGE=rex-eleve
ELEVE_CONTAINER=rex-eleve
ELEVE_CONFIG=$(RUN_DIR)/config-eleve.yaml
SECRETS_FILE=.vscode/secrets-dev.env


.PHONY: infra db-to-code docker liquibase clean start stop  start-admin stop-admin start-eleve stop-eleve ldap-start ldap-reset push

all: infra db-to-code

infra: docker liquibase

docker:
	@echo "--- 🐳 Démarrage des conteneurs Docker ---"
	cd $(DOCKER_DIR) && docker compose -f compose.yaml up -d
	
	@echo "--- 🗑️ suppression de la bd db_rex ---"
	docker exec -i postgres-16.10-alpine-rex psql -U postgres -c "DROP DATABASE IF EXISTS db_rex;"
	@echo "--- 🆕 création de la bd db_rex ---"
	docker exec -i postgres-16.10-alpine-rex psql -U postgres -c "CREATE DATABASE db_rex;"
	@echo "--- 🚀restitue la bd ---"
	docker cp ./infras/devedb_backup.dump postgres-16.10-alpine-rex:/tmp/devedb_backup.dump
	@echo "--- 🚀restaure la bd ---"
	docker exec -i postgres-16.10-alpine-rex  pg_restore --no-owner -U postgres -d db_rex /tmp/devedb_backup.dump

liquibase:
	@echo "--- 🚀 Application des migrations Liquibase ---"
	mkdir -p $(INFRA_DIR)/liquibase/liquibase_libs
	cd $(INFRA_DIR)/liquibase/liquibase_libs && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true
	@echo "--- synchronize la bd et liquidbase"
	cd $(INFRA_DIR)/liquibase && ./liquibase-dev.sh changelogSyncToTag v0.0.0
	@echo "--- 🏗️ migrations pré-suppression user_id (promotion, groupe, strongbox) ---"
	cd $(INFRA_DIR)/liquibase && ./liquibase-dev.sh updateToTag pre-drop-user-id
	@echo "--- 🚀 mise à jour des strongBox ---"
	cd $(INFRA_DIR)/migration && go build
	cd $(INFRA_DIR)/migration && ./migration
	@echo "--- 🗑️ suppression user_id + mise à jour trigger ---"
	cd $(INFRA_DIR)/liquibase && ./liquibase-dev.sh update

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

start : ldap-start start-admin start-eleve 

stop : ldap-reset stop-admin stop-eleve

start-admin:
	$(RUN_DIR)/dev-start-admin.sh

stop-admin:
	@echo "--- 🛑 Arrêt du container admin ---"
	docker rm -f $(ADMIN_CONTAINER) 2>/dev/null || true

start-eleve:
	$(RUN_DIR)/dev-start-eleve.sh

stop-eleve:
	@echo "--- 🛑 Arrêt du container élève ---"
	docker rm -f $(ELEVE_CONTAINER) 2>/dev/null || true

ldap-start:
	@echo "--- 🔑 Démarrage du LDAP local ---"
	cd $(DOCKER_DIR) && docker compose up -d openldap
	@echo "--- ✅ LDAP disponible sur ldap://localhost:3890 (dc=ema,dc=fr) ---"

ldap-reset:
	@echo "--- 🗑️ Réinitialisation du LDAP local ---"
	cd $(DOCKER_DIR) && docker compose rm -sf openldap
	docker volume rm imt-rex_openldap-data imt-rex_openldap-config 2>/dev/null || true
	@echo "--- ✅ Volumes supprimés — relancer 'make ldap-start' ---"

push:
	$(RUN_DIR)/push.sh

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose  down
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs


