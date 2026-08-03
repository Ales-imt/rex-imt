#!/bin/bash
set -euo pipefail

# Migre le contenu DÉJÀ PRÉSENT dans POSTGRES_DB : aucune donnée n'est importée ni supprimée,
# le script se contente d'appliquer les migrations Liquibase et la migration des strongBox.
#
# pour migrer la bd, se conecter avec le compte : vecu-etudiant-eleves-2.mines-ales.fr
#   ssh -L 5433:sql2.mines-ales.fr:5433 userdfx@vecu-etudiant-eleves-2.mines-ales.fr
#
# Usage: migrate-v0-to-v1.sh <secrets-file>

SECRETS_FILE="${1:-}"

if [[ -z "$SECRETS_FILE" ]]; then
    echo "Usage: $0 <secrets-file>"
    exit 1
fi

SECRETS_FILE="$(realpath "$SECRETS_FILE")"
INFRA_DIR="$(cd "$(dirname "$0")" && pwd)"
get() { grep "^$1=" "$SECRETS_FILE" | cut -d= -f2; }

POSTGRES_HOST=$(get POSTGRES_HOST)
POSTGRES_PORT=$(get POSTGRES_PORT)
POSTGRES_USER=$(get POSTGRES_USER)
POSTGRES_PASSWORD=$(get POSTGRES_PASSWORD)
POSTGRES_DB=$(get POSTGRES_DB)
AGE_PUBLIC_KEY=$(get AGE_PUBLIC_KEY)

export PGPASSWORD="$POSTGRES_PASSWORD"

echo "--- 🔍 Vérification de l'accès à la BD $POSTGRES_DB ---"
psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "\conninfo" || {
    echo "❌ Impossible de se connecter à $POSTGRES_DB"
    exit 1
}
echo "✅ BD $POSTGRES_DB accessible"

LIQUIBASE_OPTS="--changelog-file=db.changelog-master.yaml --url=jdbc:postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB --username=$POSTGRES_USER --password=$POSTGRES_PASSWORD --driver=org.postgresql.Driver"

echo "--- 🚀 Application des migrations Liquibase ---"
mkdir -p "$INFRA_DIR/liquibase/liquibase_libs"
cd "$INFRA_DIR/liquibase/liquibase_libs" && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true
echo "--- 🏗️ Création du schéma v0.0.0 ---"
# Sans restauration du dump, le schéma V0 n'existe pas : on joue réellement les
# changesets v0.0.0 (createTable) au lieu de les marquer comme appliqués.
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS updateToTag v0.0.0
echo "--- 🏗️ Migrations pré-suppression user_id ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS updateToTag pre-drop-user-id
echo "--- 🚀 Mise à jour des strongBox ---"
MIGRATION_CONFIG="$INFRA_DIR/migration/config-runtime.yaml"
sed \
    -e "s|\${POSTGRES_HOST}|$POSTGRES_HOST|g" \
    -e "s|\${POSTGRES_PORT}|$POSTGRES_PORT|g" \
    -e "s|\${POSTGRES_USER}|$POSTGRES_USER|g" \
    -e "s|\${POSTGRES_PASSWORD}|$POSTGRES_PASSWORD|g" \
    -e "s|\${POSTGRES_DB}|$POSTGRES_DB|g" \
    -e "s|\${AGE_PUBLIC_KEY}|$AGE_PUBLIC_KEY|g" \
    "$INFRA_DIR/migration/config-prod.yaml" > "$MIGRATION_CONFIG"
cd "$INFRA_DIR/migration" && go build
cd "$INFRA_DIR/migration" && ./migration --config config-runtime.yaml
rm -f "$MIGRATION_CONFIG"
echo "--- 🗑️ Suppression user_id + mise à jour trigger ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS update
