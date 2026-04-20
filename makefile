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

liquibase:
	@echo "--- 🚀 Application des migrations Liquibase ---"
	mkdir -p $(INFRA_DIR)/liquibase/liquibase_libs
	cd $(INFRA_DIR)/liquibase/liquibase_libs && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true
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
	cd $(DOCKER_DIR) && docker compose  down -v
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs


