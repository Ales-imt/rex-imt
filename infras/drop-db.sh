#!/bin/bash
set -euo pipefail

# Vide complètement la base PostgreSQL cible (schéma public + schéma migration +
# tables Liquibase) pour repartir d'une base vierge avant migrate-v0-to-v1.sh.
#
# DANGER : la suppression est DÉFINITIVE. Réservé au premier déploiement.
# La base MariaDB (cybernotes) n'est PAS touchée : c'est une base externe en lecture seule.
#
# Usage: drop-db.sh <secrets-file>
#   CONFIRM_DROP=SUPPRIMER drop-db.sh <secrets-file>  (mode non interactif)

SECRETS_FILE="${1:-}"

if [[ -z "$SECRETS_FILE" ]]; then
    echo "Usage: $0 <secrets-file>"
    exit 1
fi

SECRETS_FILE="$(realpath "$SECRETS_FILE")"
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

psql_db() { psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"; }

echo "--- 🔍 Vérification de l'accès à la BD $POSTGRES_DB ---"
psql_db -c "\conninfo" || {
    echo "❌ Impossible de se connecter à $POSTGRES_DB"
    exit 1
}
echo "✅ BD $POSTGRES_DB accessible"

TABLE_COUNT=$(psql_db -tAc "SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public';")

if [[ "$TABLE_COUNT" -gt 0 ]]; then
    echo ""
    echo "⚠️  ATTENTION — $TABLE_COUNT table(s) existent dans le schéma public de '$POSTGRES_DB'."
    echo "    Cible : $POSTGRES_USER@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB"
    echo "    La suppression est DÉFINITIVE et irréversible."
    echo "    Toutes les données seront perdues."
    echo ""
    CONFIRMATION="${CONFIRM_DROP:-}"
    if [[ -z "$CONFIRMATION" ]]; then
        read -r -p "    Tapez 'SUPPRIMER' pour confirmer : " CONFIRMATION
    fi
    if [[ "$CONFIRMATION" != "SUPPRIMER" ]]; then
        echo "❌ Annulé."
        exit 1
    fi
fi

echo "--- 🗑️ Suppression du contenu de $POSTGRES_DB ---"
psql_db -v ON_ERROR_STOP=1 <<'SQL'
DROP SCHEMA IF EXISTS migration CASCADE;
DROP TABLE IF EXISTS databasechangelog, databasechangeloglock CASCADE;
DO $$ DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
    FOR r IN (SELECT viewname FROM pg_views WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP VIEW IF EXISTS ' || quote_ident(r.viewname) || ' CASCADE';
    END LOOP;
    FOR r IN (SELECT sequencename FROM pg_sequences WHERE schemaname = 'public') LOOP
        EXECUTE 'DROP SEQUENCE IF EXISTS ' || quote_ident(r.sequencename) || ' CASCADE';
    END LOOP;
    -- Fonctions et triggers applicatifs (ex. notify_new_feedback)
    FOR r IN (SELECT p.oid::regprocedure AS sig
              FROM pg_proc p
              JOIN pg_namespace n ON n.oid = p.pronamespace
              WHERE n.nspname = 'public' AND p.prokind IN ('f', 'p')) LOOP
        EXECUTE 'DROP ROUTINE IF EXISTS ' || r.sig || ' CASCADE';
    END LOOP;
    FOR r IN (SELECT t.typname
              FROM pg_type t
              JOIN pg_namespace n ON n.oid = t.typnamespace
              WHERE n.nspname = 'public' AND t.typtype = 'e') LOOP
        EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
    END LOOP;
END $$;
SQL
echo "✅ BD $POSTGRES_DB vidée"
