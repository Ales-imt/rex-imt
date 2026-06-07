#!/bin/sh
set -e

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
SECRETS_FILE=.vscode/secrets-local.env
ADMIN_CONFIG=./infras/run/config-admin.yaml

echo "--- 📋 Copie de la configuration admin ---"
cp ./backend/admin/cmd/config.yaml "$ADMIN_CONFIG"
sed -i 's|caCertPath:.*|caCertPath: /opt/rex-admin/certs/ca.crt|' "$ADMIN_CONFIG"

echo "--- 🚀 Lancement du container admin ---"
docker rm -f rex-admin 2>/dev/null || true
docker run -d \
    --name rex-admin \
    -p 8121:8080 \
    -p 2345:2345 \
    --add-host=host.docker.internal:host-gateway \
    --env-file "$SECRETS_FILE" \
    --env HOST="$HOST_IP" \
    -v "$(pwd)/$ADMIN_CONFIG":/opt/rex-admin/conf/config.yaml:ro \
    rex-admin

echo "--- ✅ Container admin disponible sur http://localhost:8121 ---"
