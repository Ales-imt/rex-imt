#!/bin/sh
set -e

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
SECRETS_FILE=.vscode/secrets-dev.env
ELEVE_CONFIG=./infras/run/config-eleve.yaml

echo "--- 📋 Copie de la configuration élève ---"
cp ./backend/student/cmd/config.yaml "$ELEVE_CONFIG"

echo "--- 🚀 Lancement du container élève ---"
docker rm -f rex-eleve 2>/dev/null || true
docker run -d \
    --name rex-eleve \
    -p 8131:8080 \
    -p 2346:2346 \
    --add-host=host.docker.internal:host-gateway \
    --add-host=webdfd.mines-ales.fr:host-gateway \
    --env-file "$SECRETS_FILE" \
    --env HOST="$HOST_IP" \
    -v "$(pwd)/$ELEVE_CONFIG":/opt/rex-eleve/conf/config.yaml:ro \
    rex-eleve

echo "--- ✅ Container élève disponible sur http://localhost:8131 ---"
