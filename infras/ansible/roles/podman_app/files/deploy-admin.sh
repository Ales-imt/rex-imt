#!/bin/sh
set -e

# ── Configuration ────────────────────────────────────────────────────────────
GHCR_IMAGE="ghcr.io/ales-imt/rex-admin:latest"
CONTAINER_NAME="rex-admin"
SECRETS_FILE="$(dirname "$0")/secrets.env"
CONFIG_FILE="$(dirname "$0")/config-admin.yaml"
PORT="8121"

# ── Récupération du token GHCR ────────────────────────────────────────────────
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d= -f2)
if [ -z "$GITHUB_DOCKER_TOKEN" ]; then
    echo "❌ GITHUB_DOCKER_TOKEN manquant dans $SECRETS_FILE"
    exit 1
fi

# ── IP de l'hôte ─────────────────────────────────────────────────────────────
HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
echo "🖥  IP hôte : $HOST_IP"

# ── Authentification GHCR ────────────────────────────────────────────────────
echo "🔑 Connexion à ghcr.io..."
echo "$GITHUB_DOCKER_TOKEN" | docker login ghcr.io -u ales-imt --password-stdin

# ── Pull de la nouvelle image ─────────────────────────────────────────────────
echo "📥 Pull de $GHCR_IMAGE..."
docker pull "$GHCR_IMAGE"

# ── Suppression de l'ancien container ────────────────────────────────────────
echo "🗑  Suppression de l'ancien container $CONTAINER_NAME..."
podman rm --force --storage "$CONTAINER_NAME" 2>/dev/null || true
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# ── Lancement du nouveau container ───────────────────────────────────────────
echo "🚀 Lancement du container $CONTAINER_NAME..."
docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$PORT":8080 \
    --add-host=host.docker.internal:host-gateway \
    --env HOST="$HOST_IP" \
    -v "$(realpath "$CONFIG_FILE")":/opt/rex-admin/conf/config.yaml:ro,z \
    "$GHCR_IMAGE"

echo "✅ Container admin disponible sur http://localhost:$PORT"
