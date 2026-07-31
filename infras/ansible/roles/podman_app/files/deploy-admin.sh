#!/bin/sh
set -e

# Redéploiement MANUEL (fallback) de l'admin sur la VM, hors Ansible/systemd.
# Miroir simplifié du déploiement Quadlet : conteneur admin + sidecar Gotenberg
# (conversion docx -> PDF des bulletins). Gotenberg n'existe QUE sur la VM admin.
#
# NB : le prod nominal est géré par systemd/Quadlet (Restart=always). Si les
# unités systemd tournent, préférer `systemctl --user restart rex-admin` ;
# ce script est un dépannage rapide qui lance les conteneurs hors systemd.

# ── Configuration ────────────────────────────────────────────────────────────
GHCR_IMAGE="ghcr.io/ales-imt/rex-admin:latest"
CONTAINER_NAME="rex-admin"
PORT="8121"

GOTENBERG_IMAGE="docker.io/gotenberg/gotenberg:8"
GOTENBERG_NAME="gotenberg"
GOTENBERG_PORT="3000"

SECRETS_FILE="$(dirname "$0")/secrets.env"
CONFIG_FILE="$(dirname "$0")/config-admin.yaml"
FREETSA_CERT="$(dirname "$0")/x509/freetsa/cacert.pem"

# ── Récupération du token GHCR ────────────────────────────────────────────────
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d= -f2)
if [ -z "$GITHUB_DOCKER_TOKEN" ]; then
    echo "❌ GITHUB_DOCKER_TOKEN manquant dans $SECRETS_FILE"
    exit 1
fi

# ── IP de l'hôte (attendue par l'app via la variable HOST) ────────────────────
HOST_IP=$(ip route get 1 | awk '{print $7; exit}')
echo "🖥  IP hôte : $HOST_IP"

# ── Sidecar Gotenberg (conversion docx -> PDF) ────────────────────────────────
# Publié sur l'hôte : l'app admin le joint via host.containers.internal:3000
# (GOTENBERG_URL dans secrets.env). Exposition contenue par le firewall de la VM.
echo "📥 Pull de $GOTENBERG_IMAGE..."
podman pull "$GOTENBERG_IMAGE"

echo "🗑  Suppression de l'ancien conteneur $GOTENBERG_NAME..."
podman rm --force "$GOTENBERG_NAME" 2>/dev/null || true

echo "🚀 Lancement de $GOTENBERG_NAME (port $GOTENBERG_PORT)..."
podman run -d \
    --name "$GOTENBERG_NAME" \
    --restart unless-stopped \
    -p "$GOTENBERG_PORT":3000 \
    "$GOTENBERG_IMAGE"

# ── Conteneur admin ───────────────────────────────────────────────────────────
FREETSA_VOLUME=""
if [ -f "$FREETSA_CERT" ]; then
    FREETSA_VOLUME="-v $(realpath "$FREETSA_CERT"):/opt/rex-admin/conf/x509/freetsa/cacert.pem:ro,z"
else
    echo "--- ⚠️  Cert FreeTSA absent ($FREETSA_CERT) — ancrage TSA désactivé."
fi

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
    -v "$(realpath "$CONFIG_FILE")":/opt/rex-admin/conf/config.yaml:ro,z \
    $FREETSA_VOLUME \
    "$GHCR_IMAGE"

echo "✅ Admin sur http://localhost:$PORT — Gotenberg sur http://localhost:$GOTENBERG_PORT"
