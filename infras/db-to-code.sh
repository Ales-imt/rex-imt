#!/bin/bash
set -euo pipefail

# Usage: db-to-code.sh <secrets-file>

SECRETS_FILE="${1:-}"

if [[ -z "$SECRETS_FILE" ]]; then
    echo "Usage: $0 <secrets-file>"
    exit 1
fi

SECRETS_FILE="$(realpath "$SECRETS_FILE")"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
get() { grep "^$1=" "$SECRETS_FILE" | cut -d= -f2; }

POSTGRES_HOST=$(get POSTGRES_HOST)
POSTGRES_PORT=$(get POSTGRES_PORT)
POSTGRES_USER=$(get POSTGRES_USER)
POSTGRES_PASSWORD=$(get POSTGRES_PASSWORD)
POSTGRES_DB=$(get POSTGRES_DB)

MARIADB_HOST=$(get MARIADB_HOST)
MARIADB_PORT=$(get MARIADB_PORT)
MARIADB_USER=$(get MARIADB_USER)
MARIADB_PASSWORD=$(get MARIADB_PASSWORD)
MARIADB_DB=$(get MARIADB_DB)

DB_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB"

BACK_DIR="$PROJECT_ROOT/backend"
SCHEMA_FILE_POSTGRES="schema.sql"
SCHEMA_FILE_MARIADB="schema_maria_db.sql"

echo "--- 🐘 1. Extraction du schéma PostgreSQL ---"
pg_dump -s -x -O -d "$DB_URL" -T "databasechangelog*" > "$BACK_DIR/$SCHEMA_FILE_POSTGRES"

echo "--- 🐬 Extraction du schéma MariaDB ---"
docker run --rm --network=host mariadb:latest \
    mariadb-dump -h "$MARIADB_HOST" -P "$MARIADB_PORT" \
    -u "$MARIADB_USER" -p"$MARIADB_PASSWORD" \
    --skip-ssl --skip-lock-tables --no-data "$MARIADB_DB" > "$BACK_DIR/$SCHEMA_FILE_MARIADB"

echo "--- 🧹 2. Nettoyage du fichier SQL ---"
sed -i '/restrict/d' "$BACK_DIR/$SCHEMA_FILE_POSTGRES"
sed -i '/unrestrict/d' "$BACK_DIR/$SCHEMA_FILE_POSTGRES"
sed -i '/^--/d' "$BACK_DIR/$SCHEMA_FILE_POSTGRES"

echo "--- 🏗️ 3. Lancement de sqlc generate ---"
(cd "$BACK_DIR/admin"   && sqlc generate)
(cd "$BACK_DIR/common"  && sqlc generate)
(cd "$BACK_DIR/student" && sqlc generate)
