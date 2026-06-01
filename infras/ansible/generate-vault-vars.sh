#!/bin/sh
set -e

SECRETS="$(dirname "$0")/../../.vscode/secrets-prod.env"
OUTPUT="$(dirname "$0")/vault-vars.yml"

get() { grep "^$1=" "$SECRETS" | cut -d= -f2; }

cat > "$OUTPUT" <<EOF
---
github_docker_token: "$(get GITHUB_DOCKER_TOKEN)"

postgres_user: "$(get POSTGRES_USER)"
postgres_password: "$(get POSTGRES_PASSWORD)"
postgres_host: "$(get POSTGRES_HOST)"
postgres_port: $(get POSTGRES_PORT)
postgres_db: "$(get POSTGRES_DB)"

mariadb_host: "$(get MARIADB_HOST)"
mariadb_port: $(get MARIADB_PORT)
mariadb_database: "$(get MARIADB_DB)"
mariadb_user: "$(get MARIADB_USER)"
mariadb_password: "$(get MARIADB_PASSWORD)"

jwt_secret: "$(get JWT_SECRET)"

rack_api_key: "$(get RACK_API_KEY)"

EOF

echo "✅ vault-vars.yml généré — chiffre avec : ansible-vault encrypt vault-vars.yml"
