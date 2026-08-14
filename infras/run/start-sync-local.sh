#!/bin/sh
set -e

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
CONFIG_FILE=infras/env/config-local.env
SECRETS_FILE=infras/env/secrets-local.env

# Un fichier absent ne doit pas passer inaperçu : les valeurs manquantes
# deviennent des chaînes VIDES au moment de résoudre les ${VAR}, et le
# déploiement réussirait avec une configuration creuse.
for _f in "$CONFIG_FILE" "$SECRETS_FILE"; do
    [ -f "$_f" ] || { echo "❌ Fichier de configuration absent : $_f" >&2
                      echo "   Voir infras/env/README.md" >&2; exit 1; }
done

SYNC_CONFIG=./infras/run/config-sync.yaml

echo "--- 📋 Copie de la configuration sync ---"
cp ./backend/sync/cmd/config.yaml "$SYNC_CONFIG"

# Le répertoire des exports est monté au MÊME chemin que sur l'hôte : la config
# est la même source pour les deux, une seule réécriture suffirait sinon.
# Les chemins hfsql viennent de la topologie locale ; une variable
# d'environnement les surcharge ponctuellement.
_conf() { grep -h "^$1=" "$CONFIG_FILE" 2>/dev/null | head -1 | cut -d= -f2-; }
HFSQL_DATA="${HFSQL_DATA:-$(_conf HFSQL_DATA)}"
sed -i "s|data: /var/lib/rex/hfsql/data|data: $HFSQL_DATA|" "$SYNC_CONFIG"

# Le seul secret dont ce service a besoin est le mot de passe PostgreSQL : il
# est passé nommément plutôt que par --env-file, qui lui livrerait aussi les
# clés JWT, les clés d'API et le jeton GHCR. C'est ce que le découpage
# config/secrets rend possible.
#
# Adressage statique de rex-net (compose.yaml + scripts de lancement) :
#   .2 ollama   .4 postgres   .5 openldap   .6 mariadb   .7 gotenberg
#   .10 admin   .11 eleve     .12 sync
# export-webdfd n'y figure pas : il ne parle à personne sur le réseau.
echo "--- 🚀 Lancement du container sync ---"
docker rm -f rex-sync 2>/dev/null || true
docker run -d \
    --name rex-sync \
    --network imt-rex_rex-net \
    --ip 10.20.1.12 \
    --add-host=host.docker.internal:host-gateway \
    --env-file "$CONFIG_FILE" \
    -e POSTGRES_PASSWORD="$(grep -h '^POSTGRES_PASSWORD=' "$SECRETS_FILE" | cut -d= -f2-)" \
    -e HTTP_PROXY=socks5h://host.docker.internal:1080 \
    -e NO_PROXY=10.20.1.4,10.20.1.5,10.20.1.6,localhost,127.0.0.1 \
    -v "$(pwd)/$SYNC_CONFIG":/opt/rex-sync/conf/config.yaml:ro \
    -v "$HFSQL_DATA":"$HFSQL_DATA":ro \
    rex-sync

echo "--- ✅ Container sync démarré (santé : http://10.20.1.12:3334/health) ---"
