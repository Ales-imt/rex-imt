#!/bin/sh
set -e

SECRETS="$(dirname "$0")/../../.vscode/secrets-prod.env"
OUTPUT="$(dirname "$0")/vault-vars.yml"

# -f2- et non -f2 : une valeur contenant un « = » (padding base64 d'un secret
# régénéré, par exemple) serait sinon tronquée silencieusement.
get() { grep "^$1=" "$SECRETS" | cut -d= -f2-; }

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

ldap_host: "$(get LDAP_HOST)"

jwt_secret_admin: "$(get JWT_SECRET_ADMIN)"
jwt_secret_eleve: "$(get JWT_SECRET_ELEVE)"

rack_api_key: "$(get RACK_API_KEY)"

age_public_key: "$(get AGE_PUBLIC_KEY)"

presence_token_secret: "$(get PRESENCE_TOKEN_SECRET)"

# Témoin externe des ancrages de présence (config.yaml : presence.witness).
# Sans ces trois variables, smtp.host arrive vide côté serveur et witnessEnabled()
# désactive l'envoi en silence, alors que witness.enabled vaut true.
witness_recipient: "$(get WITNESS_RECIPIENT)"
smtp_host: "$(get SMTP_HOST)"
smtp_port: $(get SMTP_PORT)

EOF

