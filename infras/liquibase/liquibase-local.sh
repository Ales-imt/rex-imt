#!/bin/sh
set -e

SECRETS_FILE="$(dirname "$0")/../../.vscode/secrets-dev.env"
get() { grep "^$1=" "$SECRETS_FILE" | cut -d= -f2; }

POSTGRES_HOST=$(get POSTGRES_HOST)
POSTGRES_PORT=$(get POSTGRES_PORT)
POSTGRES_USER=$(get POSTGRES_USER)
POSTGRES_PASSWORD=$(get POSTGRES_PASSWORD)
POSTGRES_DB=$(get POSTGRES_DB)

LIQUIBASE_OPTS="--changelog-file=db.changelog-master.yaml --url=jdbc:postgresql://$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB --username=$POSTGRES_USER --password=$POSTGRES_PASSWORD --driver=org.postgresql.Driver"

cd "$(dirname "$0")"
liquibase $LIQUIBASE_OPTS "$@"
