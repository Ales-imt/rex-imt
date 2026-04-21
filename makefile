# --- Variables ---
DB_URL=postgres://postgres:root@localhost:5432/db_rex
SCHEMA_FILE=schema.sql
BACK_DIR=./backend
INFRA_DIR=./infras
DOCKER_DIR=./infras/container

.PHONY: infra db-to-code docker liquibase clean

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
	docker cp ./devedb_backup.dump postgres-16.10-alpine-rex:/tmp/devedb_backup.dump
	@echo "--- 🚀restaure la bd ---"
	docker exec -i postgres-16.10-alpine-rex  pg_restore --no-owner -U postgres -d db_rex /tmp/devedb_backup.dump

liquibase:
	@echo "--- 🚀 Application des migrations Liquibase ---"
	mkdir -p $(INFRA_DIR)/liquibase/liquibase_libs
	cd $(INFRA_DIR)/liquibase/liquibase_libs && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true
	@echo "--- synchronize la bd et liquidbase"
	cd $(INFRA_DIR)/liquibase && liquibase changelogSyncToTag v0.0.0
	@echo "--- appliquer les migrations"
	cd $(INFRA_DIR)/liquibase && liquibase update


db-to-code:
	@echo "--- 🐘 1. Extraction du schéma PostgreSQL ---"
	pg_dump -s -x -O -d $(DB_URL) -T "databasechangelog*" > $(BACK_DIR)/$(SCHEMA_FILE)
	
	@echo "--- 🧹 2. Nettoyage du fichier SQL ---"
	sed -i '/restrict/d' $(BACK_DIR)/$(SCHEMA_FILE)
	sed -i '/unrestrict/d' $(BACK_DIR)/$(SCHEMA_FILE)
	sed -i '/^--/d' $(BACK_DIR)/$(SCHEMA_FILE)
	
	@echo "--- 🏗️ 3. Lancement de sqlc generate ---"
	cd $(BACK_DIR)/admin && sqlc generate
	cd $(BACK_DIR)/common && sqlc generate
	cd $(BACK_DIR)/student && sqlc generate

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose  down  
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs


