#!/bin/sh
set -e

SECRETS_FILE=.vscode/secrets-prod.env
GHCR_REGISTRY=ghcr.io
GHCR_ORG=ales-imt
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d'=' -f2)

echo "--- ⚙️  Génération sqlc ---"
(cd backend/common  && sqlc generate)
(cd backend/admin   && sqlc generate)
(cd backend/student && sqlc generate)

echo "--- 🐳 Build de l'image admin ---"
docker build -f ./infras/run/Dockerfile.admin \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t rex-admin .

echo "--- 🐳 Build de l'image élève ---"
docker build -f ./infras/run/Dockerfile.eleve \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t rex-eleve .

echo "--- 🔑 Connexion à GHCR ---"
echo "$GITHUB_DOCKER_TOKEN" | docker login "$GHCR_REGISTRY" -u "$GHCR_ORG" --password-stdin

echo "--- 🏷️ Tag des images ---"
docker tag rex-admin  "$GHCR_REGISTRY/$GHCR_ORG/rex-admin:latest"
docker tag rex-eleve  "$GHCR_REGISTRY/$GHCR_ORG/rex-eleve:latest"

echo "--- 🚀 Push vers GHCR ---"
docker push "$GHCR_REGISTRY/$GHCR_ORG/rex-admin:latest"
docker push "$GHCR_REGISTRY/$GHCR_ORG/rex-eleve:latest"

echo "--- ✅ Images publiées sur $GHCR_REGISTRY/$GHCR_ORG ---"
