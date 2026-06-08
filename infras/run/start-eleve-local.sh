#!/bin/sh
set -e

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
SECRETS_FILE=.vscode/secrets-local.env
ELEVE_CONFIG=./infras/run/config-eleve.yaml

echo "--- 📋 Copie de la configuration élève ---"
cp ./backend/student/cmd/config.yaml "$ELEVE_CONFIG"

echo "--- 🚀 Lancement du container élève ---"
docker rm -f rex-eleve 2>/dev/null || true
docker run -d \
    --name rex-eleve \
    --network imt-rex_rex-net \
    --ip 10.20.1.11 \
    --add-host=host.docker.internal:host-gateway \
    --add-host=webdfd.mines-ales.fr:host-gateway \
    --env-file "$SECRETS_FILE" \
    -v "$(pwd)/$ELEVE_CONFIG":/opt/rex-eleve/conf/config.yaml:ro \
    rex-eleve

echo " pour aller du telephone vers le PC socat TCP-LISTEN:8131,bind=10.208.113.46,fork TCP:10.20.1.11:8131"
echo " puis http://10.208.113.46:8131"

echo "--- ✅ Container élève disponible sur http://10.20.1.11:8131 ---"
