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

.PHONY: all db-to-code clean fetch-freetsa-cert fetch-freetsa-cert-if-missing

all:
	@echo ""
	@echo "Usage : make local — déploiement local"
	@echo "        make first-deploy-prod  — déploiement production avec supression BD "
	@echo "        make release-prod  — déploiement production"
	@echo "        make fetch-freetsa-cert — télécharge le certificat racine FreeTSA"
	@echo ""

# ── Certificat FreeTSA (ancrage RFC 3161) ─────────────────────────────────────
# Télécharge le certificat racine de FreeTSA et le place à l'emplacement attendu
# par la config admin (presence.timestamp.caCertPath).
# À exécuter une fois avant le premier démarrage si l'ancrage TSA est activé.
# Vérifier l'empreinte affichée sur https://freetsa.org avant de faire confiance.

FREETSA_CERT_DIR=$(BACK_DIR)/admin/x509/freetsa
FREETSA_CERT_URL=https://freetsa.org/files/cacert.pem
FREETSA_CERT_PATH=$(FREETSA_CERT_DIR)/cacert.pem

fetch-freetsa-cert:
	@echo "--- 🔒 Téléchargement du certificat racine FreeTSA ---"
	@mkdir -p $(FREETSA_CERT_DIR)
	@curl -fsSL --output $(FREETSA_CERT_PATH) $(FREETSA_CERT_URL)
	@echo "--- ✅ Certificat enregistré dans $(FREETSA_CERT_PATH) ---"
	@echo "--- 🔍 Empreinte SHA-256 (à vérifier sur https://freetsa.org) ---"
	@openssl x509 -in $(FREETSA_CERT_PATH) -noout -fingerprint -sha256

# Télécharge uniquement si le certificat est absent (pas de requête réseau à chaque build).
# Utilisé comme prérequis du build Docker en prod.
fetch-freetsa-cert-if-missing:
	@if [ ! -f $(FREETSA_CERT_PATH) ]; then \
		$(MAKE) fetch-freetsa-cert; \
	else \
		echo "--- ✅ Certificat FreeTSA déjà présent ($(FREETSA_CERT_PATH)) ---"; \
	fi

# ── Nettoyage ─────────────────────────────────────────────────────────────────

clean:
#-v pour tous supprimer.
	cd $(DOCKER_DIR) && docker compose --env-file ../../$(SECRETS_FILE_LOCAL) down
	rm -rf $(INFRA_DIR)/liquibase/liquibase_libs
