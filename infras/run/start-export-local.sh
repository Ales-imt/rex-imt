#!/bin/sh
set -e

EXPORT_CONFIG=./infras/run/config-export-webdfd.yaml

echo "--- 📋 Copie de la configuration export-webdfd ---"
cp ./backend/export-webdfd/cmd/config.yaml "$EXPORT_CONFIG"

# En local comme en prod, le service tourne DANS l'image wine : runner local.
CONFIG_FILE=infras/env/config-local.env
[ -f "$CONFIG_FILE" ] || { echo "❌ Fichier de configuration absent : $CONFIG_FILE" >&2
                           echo "   Voir infras/env/README.md" >&2; exit 1; }

# Les chemins hfsql viennent de la topologie locale ; une variable
# d'environnement les surcharge ponctuellement.
_conf() { grep -h "^$1=" "$CONFIG_FILE" 2>/dev/null | head -1 | cut -d= -f2-; }
HFSQL_DEPOT="${HFSQL_DEPOT:-$(_conf HFSQL_DEPOT)}"
HFSQL_DATA="${HFSQL_DATA:-$(_conf HFSQL_DATA)}"
sed -i \
    -e "s|depot: /var/lib/rex/hfsql/depot|depot: $HFSQL_DEPOT|" \
    -e "s|data: /var/lib/rex/hfsql/data|data: $HFSQL_DATA|" \
    -e "s|runner: docker|runner: local|" \
    "$EXPORT_CONFIG"

mkdir -p "$HFSQL_DEPOT" "$HFSQL_DATA"

echo "--- 🚀 Lancement du container export-webdfd ---"
docker rm -f rex-export-webdfd 2>/dev/null || true
# Les répertoires sont montés au MÊME chemin qu'ils ont sur l'hôte : inutile de
# réécrire la configuration, et les messages du journal restent lisibles.
# Aucun secret, aucun port : ce service ne connaît ni base ni réseau.
docker run -d \
    --name rex-export-webdfd \
    --user "$(id -u):$(id -g)" \
    -v "$(pwd)/$EXPORT_CONFIG":/opt/rex-export/conf/config.yaml:ro \
    -v "$HFSQL_DEPOT":"$HFSQL_DEPOT" \
    -v "$HFSQL_DATA":"$HFSQL_DATA" \
    rex-export-webdfd

echo "--- ✅ Container export-webdfd démarré (surveille $HFSQL_DEPOT) ---"
