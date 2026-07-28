#!/bin/sh
set -e

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
SECRETS_FILE=.vscode/secrets-local.env
ADMIN_CONFIG=./infras/run/config-admin.yaml

echo "--- 📋 Copie de la configuration admin ---"
cp ./backend/admin/cmd/config.yaml "$ADMIN_CONFIG"
# Même réécriture de chemin que le template ansible config-admin.yaml.j2 :
# seul le caCertPath freetsa change entre la source backend et le container.
sed -i \
    -e 's|caCertPath: \./x509/freetsa/cacert\.pem|caCertPath: /opt/rex-admin/conf/x509/freetsa/cacert.pem|' \
    "$ADMIN_CONFIG"

FREETSA_CERT=./backend/admin/x509/freetsa/cacert.pem
FREETSA_VOLUME=""
if [ -f "$FREETSA_CERT" ]; then
    FREETSA_VOLUME="-v $(pwd)/$FREETSA_CERT:/opt/rex-admin/conf/x509/freetsa/cacert.pem:ro"
else
    echo "--- ⚠️  Cert FreeTSA absent ($FREETSA_CERT) — ancrage TSA désactivé. Exécuter: make fetch-freetsa-cert"
fi

echo "--- 🚀 Lancement du container admin ---"
docker rm -f rex-admin 2>/dev/null || true
docker run -d \
    --name rex-admin \
    --network imt-rex_rex-net \
    --ip 10.20.1.10 \
    --add-host=host.docker.internal:host-gateway \
    --env-file "$SECRETS_FILE" \
    -e HTTP_PROXY=socks5h://host.docker.internal:1080 \
    -e NO_PROXY=10.20.1.4,10.20.1.5,10.20.1.6,localhost,127.0.0.1 \
    -v "$(pwd)/$ADMIN_CONFIG":/opt/rex-admin/conf/config.yaml:ro \
    $FREETSA_VOLUME \
    rex-admin

echo " pour aller du telephone vers le PC socat TCP-LISTEN:8121,bind=10.208.113.46,fork TCP:10.20.1.11:8121"
echo " puis http://10.208.113.46:8121"

echo "--- ✅ Container admin disponible sur http://10.20.1.10:8121 ---"
