#!/bin/sh
set -e

# Redéploiement MANUEL (fallback) de rex-export-webdfd sur la VM, hors
# Ansible/systemd. Préférer `systemctl --user restart rex-export-webdfd`.
#
# AUCUN secret n'est passé à ce conteneur : il ne connaît ni base ni réseau.
# --userns=keep-id est indispensable — sans lui, podman rootless remappe l'UID
# et les exports produits n'appartiennent pas à l'utilisateur de service, si
# bien que rex-sync ne peut pas les lire.

GHCR_IMAGE="${GHCR_IMAGE:-ghcr.io/ales-imt/rex-export-webdfd:latest}"
CONTAINER_NAME="rex-export-webdfd"
HFSQL_DEPOT="${HFSQL_DEPOT:-/srv/hfsql}"
HFSQL_DATA="${HFSQL_DATA:-/srv/hfsql-exports}"

SECRETS_FILE="$(dirname "$0")/secrets.env"
CONFIG_FILE="$(dirname "$0")/config-export-webdfd.yaml"

# Le token GHCR est le SEUL secret nécessaire, et seulement pour tirer l'image.
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" 2>/dev/null | cut -d= -f2)
if [ -n "$GITHUB_DOCKER_TOKEN" ]; then
    echo "🔑 Connexion à ghcr.io..."
    echo "$GITHUB_DOCKER_TOKEN" | podman login ghcr.io -u ales-imt --password-stdin
fi

echo "📥 Pull de $GHCR_IMAGE..."
podman pull "$GHCR_IMAGE"

echo "🗑  Suppression de l'ancien conteneur $CONTAINER_NAME..."
podman rm --force "$CONTAINER_NAME" 2>/dev/null || true

echo "🚀 Lancement de $CONTAINER_NAME..."
podman run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    --userns=keep-id \
    -v "$(realpath "$CONFIG_FILE")":/opt/rex-export/conf/config.yaml:ro,z \
    -v "$HFSQL_DEPOT":"$HFSQL_DEPOT":ro,z \
    -v "$HFSQL_DATA":"$HFSQL_DATA":z \
    "$GHCR_IMAGE"

echo "✅ Export-webdfd démarré — surveille $HFSQL_DEPOT"
