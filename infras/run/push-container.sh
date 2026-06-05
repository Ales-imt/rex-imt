#!/bin/sh
set -e

SECRETS_FILE=.vscode/secrets-prod.env
GHCR_REGISTRY=ghcr.io
GHCR_ORG=ales-imt
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d'=' -f2)

echo "--- ⚙️  Génération sqlc ---"
(cd backend/common  && sqlc generate)
(cd backend/admin   && sqlc generate)
(cd backend/student && sqlc generate)

./infras/run/build-admin.sh
./infras/run/build-eleve.sh

echo "--- 🔑 Connexion à GHCR ---"
echo "$GITHUB_DOCKER_TOKEN" | docker login "$GHCR_REGISTRY" -u "$GHCR_ORG" --password-stdin

echo "--- 🏷️ Tag des images ---"
docker tag rex-admin  "$GHCR_REGISTRY/$GHCR_ORG/rex-admin:latest"
docker tag rex-eleve  "$GHCR_REGISTRY/$GHCR_ORG/rex-eleve:latest"

echo "--- 🚀 Push vers GHCR ---"
docker push "$GHCR_REGISTRY/$GHCR_ORG/rex-admin:latest"
docker push "$GHCR_REGISTRY/$GHCR_ORG/rex-eleve:latest"

echo "--- ✅ Images publiées sur $GHCR_REGISTRY/$GHCR_ORG ---"
