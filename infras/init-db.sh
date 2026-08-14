#!/bin/bash
set -euo pipefail

# Crée le schéma complet dans POSTGRES_DB en jouant tous les changesets Liquibase.
#
# Aucune donnée de la V0.0.0 n'est reprise : pas de restauration de dump, pas de
# migration des strongBox. Le schéma est construit de zéro par Liquibase.
#
# Usage: init-db.sh <secrets-file>

SECRETS_FILE="${1:-}"

if [[ -z "$SECRETS_FILE" ]]; then
    echo "Usage: $0 <secrets-file>"
    exit 1
fi

SECRETS_FILE="$(realpath "$SECRETS_FILE")"
INFRA_DIR="$(cd "$(dirname "$0")" && pwd)"
# La configuration est scindée en deux : la topologie (config-*.env) et les
# secrets (secrets-*.env). Le chemin du second est donné en argument, le premier
# s'en deduit par convention de nommage. `-f2-` et non `-f2` : une valeur
# contenant un « = » (padding base64) serait sinon tronquee en silence.
CONFIG_FILE="${SECRETS_FILE/secrets-/config-}"
get() { grep -h "^$1=" "$CONFIG_FILE" "$SECRETS_FILE" 2>/dev/null | head -1 | cut -d= -f2-; }

POSTGRES_HOST=$(get POSTGRES_HOST)
POSTGRES_PORT=$(get POSTGRES_PORT)
POSTGRES_USER=$(get POSTGRES_USER)
POSTGRES_PASSWORD=$(get POSTGRES_PASSWORD)
POSTGRES_DB=$(get POSTGRES_DB)

export PGPASSWORD="$POSTGRES_PASSWORD"

echo "--- 🔍 Vérification de l'accès à la BD $POSTGRES_DB ---"
psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "\conninfo" || {
    echo "❌ Impossible de se connecter à $POSTGRES_DB"
    exit 1
}
echo "✅ BD $POSTGRES_DB accessible"

LIQUIBASE_OPTS="--changelog-file=db.changelog-master.yaml --url=jdbc:postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB --username=$POSTGRES_USER --password=$POSTGRES_PASSWORD --driver=org.postgresql.Driver"

echo "--- 📦 Driver PostgreSQL pour Liquibase ---"
mkdir -p "$INFRA_DIR/liquibase/liquibase_libs"
cd "$INFRA_DIR/liquibase/liquibase_libs" && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true

echo "--- 🏗️ Création du schéma (tous les changesets) ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS update
echo "✅ Schéma créé dans $POSTGRES_DB"
