#!/bin/sh
set -e

SECRETS_FILE="$(dirname "$0")/../env/secrets-local.env"
CONFIG_FILE="$(dirname "$0")/../env/config-local.env"
get() { grep -h "^$1=" "$CONFIG_FILE" "$SECRETS_FILE" 2>/dev/null | head -1 | cut -d= -f2-; }

POSTGRES_HOST=$(get POSTGRES_HOST)
POSTGRES_PORT=$(get POSTGRES_PORT)
POSTGRES_USER=$(get POSTGRES_USER)
POSTGRES_PASSWORD=$(get POSTGRES_PASSWORD)
POSTGRES_DB=$(get POSTGRES_DB)

LIQUIBASE_OPTS="--changelog-file=db.changelog-master.yaml --url=jdbc:postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB --username=$POSTGRES_USER --password=$POSTGRES_PASSWORD --driver=org.postgresql.Driver"

cd "$(dirname "$0")"
liquibase $LIQUIBASE_OPTS "$@"
