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
    --network imt-rex_rex-net \
    --ip 10.20.1.10 \
    --add-host=host.docker.internal:host-gateway \
    --env-file "$SECRETS_FILE" \
    -v "$(pwd)/$ADMIN_CONFIG":/opt/rex-admin/conf/config.yaml:ro \
    rex-admin

echo " pour aller du telephone vers le PC socat TCP-LISTEN:8121,bind=10.208.113.46,fork TCP:10.20.1.11:8121"
echo " puis http://10.208.113.46:8121"

echo "--- ✅ Container admin disponible sur http://10.20.1.10:8121 ---"
