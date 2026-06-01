#!/bin/sh
set -e

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
SECRETS_FILE=.vscode/secrets-dev.env
ADMIN_CONFIG=./infras/run/config-admin.yaml

echo "--- 📋 Copie de la configuration admin ---"
cp ./backend/admin/cmd/config.yaml "$ADMIN_CONFIG"
sed -i 's|caCertPath:.*|caCertPath: /opt/rex-admin/certs/ca.crt|' "$ADMIN_CONFIG"

echo "--- 🐳 Build de l'image admin ---"
docker build -f ./infras/run/Dockerfile.admin \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t rex-admin .

echo "--- 🚀 Lancement du container admin ---"
docker rm -f rex-admin 2>/dev/null || true
docker run -d \
    --name rex-admin \
    -p 8121:8080 \
    --add-host=host.docker.internal:host-gateway \
    --env-file "$SECRETS_FILE" \
    --env HOST="$HOST_IP" \
    -v "$(pwd)/$ADMIN_CONFIG":/opt/rex-admin/conf/config.yaml:ro \
    rex-admin

echo "--- ✅ Container admin disponible sur http://localhost:8121 ---"
