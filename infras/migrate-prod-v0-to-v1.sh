#/bin/bash
set -euo pipefail

# dump de la bd V0, obligé de se logger avec le compte : userde@vecu-etudiant.mines-ales.fr 
#   ssh -L 5433:sql2.mines-ales.fr:5433 userde@vecu-etudiant.mines-ales.fr
#   pg_dump -h localhost -p 5433  -U devedbuserext  -d devedb -F c -f devedb_backup.dump

# pour migrer de la bd V0 dans la nouvelle bd, se conecter avec le compte : vecu-etudiant-eleves-2.mines-ales.fr
#   ssh -L 5433:sql2.mines-ales.fr:5433 userdfx@vecu-etudiant-eleves-2.mines-ales.fr

INFRA_DIR="$(cd "$(dirname "$0")" && pwd)"

# ── Connexion BD prod ─────────────────────────────────────────────────────────
SECRETS_FILE="$(dirname "$0")/../.vscode/secrets-prod.env"
get() { grep "^$1=" "$SECRETS_FILE" | cut -d= -f2; }

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

echo "--- 🗑️ Suppression des tables existantes de $POSTGRES_DB ---"
psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" <<'SQL'
DO $$ DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END $$;
SQL
echo "✅ Tables supprimées"

echo "--- 📥 Restauration du dump V0 dans $POSTGRES_DB ---"
pg_restore -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    --no-owner --no-privileges \
    ./devedb_backup.dump
echo "✅ Dump restauré"

LIQUIBASE_OPTS="--changelog-file=db.changelog-master.yaml --url=jdbc:postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB --username=$POSTGRES_USER --password=$POSTGRES_PASSWORD --driver=org.postgresql.Driver"

echo "--- 🚀 Application des migrations Liquibase ---"
mkdir -p "$INFRA_DIR/liquibase/liquibase_libs"
cd "$INFRA_DIR/liquibase/liquibase_libs" && wget -nc https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.8/postgresql-42.7.8.jar || true
echo "--- 🔄 Synchronisation Liquibase ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS changelogSyncToTag v0.0.0
echo "--- 🏗️ Migrations pré-suppression user_id ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS updateToTag pre-drop-user-id
echo "--- 🚀 Mise à jour des strongBox ---"
MIGRATION_CONFIG="$INFRA_DIR/migration/config-prod-runtime.yaml"
sed \
    -e "s|\${POSTGRES_HOST}|$POSTGRES_HOST|g" \
    -e "s|\${POSTGRES_PORT}|$POSTGRES_PORT|g" \
    -e "s|\${POSTGRES_USER}|$POSTGRES_USER|g" \
    -e "s|\${POSTGRES_PASSWORD}|$POSTGRES_PASSWORD|g" \
    -e "s|\${POSTGRES_DB}|$POSTGRES_DB|g" \
    "$INFRA_DIR/migration/config-prod.yaml" > "$MIGRATION_CONFIG"
cd "$INFRA_DIR/migration" && go build
cd "$INFRA_DIR/migration" && ./migration --config config-prod-runtime.yaml
rm -f "$MIGRATION_CONFIG"
echo "--- 🗑️ Suppression user_id + mise à jour trigger ---"
cd "$INFRA_DIR/liquibase" && liquibase $LIQUIBASE_OPTS update

