#!/bin/sh
set -e

# Pousse UNE image déjà construite localement vers GHCR.
#
# push-container.sh reconstruit et repousse les quatre images du projet : c'est
# ce qu'il faut pour une release complète, mais pas pour corriger un seul
# service — l'image export-webdfd pèse à elle seule ~2 Go.
#
# Usage : push-image.sh <nom-image> [tag]

SECRETS_FILE=infras/env/secrets-prod.env
GHCR_REGISTRY=ghcr.io
GHCR_ORG=ales-imt

IMAGE="${1:?nom de l'image attendu (ex: rex-export-webdfd)}"
IMAGE_TAG="${2:-${IMAGE_TAG:-dev}}"

# -f2- et non -f2 : un token contenant un « = » serait sinon tronqué en silence.
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d= -f2-)

echo "--- 🔑 Connexion à GHCR ---"
echo "$GITHUB_DOCKER_TOKEN" | docker login "$GHCR_REGISTRY" -u "$GHCR_ORG" --password-stdin

echo "--- 🏷️ Tag de $IMAGE ($IMAGE_TAG) ---"
docker tag "$IMAGE:$IMAGE_TAG" "$GHCR_REGISTRY/$GHCR_ORG/$IMAGE:$IMAGE_TAG"

echo "--- 🚀 Push de $IMAGE ---"
docker push "$GHCR_REGISTRY/$GHCR_ORG/$IMAGE:$IMAGE_TAG"

echo "--- ✅ $GHCR_REGISTRY/$GHCR_ORG/$IMAGE:$IMAGE_TAG publiée ---"
