#!/bin/sh
set -e

# Redéploiement MANUEL (fallback) de rex-sync sur la VM, hors Ansible/systemd.
#
# NB : le prod nominal est géré par systemd/Quadlet (Restart=always). Si les
# unités tournent, préférer `systemctl --user restart rex-sync` ; ce script est
# un dépannage rapide qui lance le conteneur hors systemd.

GHCR_IMAGE="${GHCR_IMAGE:-ghcr.io/ales-imt/rex-sync:latest}"
CONTAINER_NAME="rex-sync"
PORT="3334"
HFSQL_DATA="${HFSQL_DATA:-/srv/hfsql-exports}"

SECRETS_FILE="$(dirname "$0")/secrets.env"
CONFIG_FILE="$(dirname "$0")/config-sync.yaml"

GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d= -f2)
if [ -z "$GITHUB_DOCKER_TOKEN" ]; then
    echo "❌ GITHUB_DOCKER_TOKEN manquant dans $SECRETS_FILE"
    exit 1
fi

HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
echo "🖥  IP hôte : $HOST_IP"

echo "🔑 Connexion à ghcr.io..."
echo "$GITHUB_DOCKER_TOKEN" | podman login ghcr.io -u ales-imt --password-stdin

echo "📥 Pull de $GHCR_IMAGE..."
podman pull "$GHCR_IMAGE"

echo "🗑  Suppression de l'ancien conteneur $CONTAINER_NAME..."
podman rm --force "$CONTAINER_NAME" 2>/dev/null || true

echo "🚀 Lancement de $CONTAINER_NAME..."
podman run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$PORT":"$PORT" \
    --add-host=host.containers.internal:host-gateway \
    --env HOST="$HOST_IP" \
    --env-file "$SECRETS_FILE" \
    -v "$(realpath "$CONFIG_FILE")":/opt/rex-sync/conf/config.yaml:ro,z \
    -v "$HFSQL_DATA":"$HFSQL_DATA":ro,z \
    "$GHCR_IMAGE"

echo "✅ Sync démarré — santé : http://localhost:$PORT/health"
