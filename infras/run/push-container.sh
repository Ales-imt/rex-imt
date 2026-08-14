#!/bin/sh
set -e

SECRETS_FILE=infras/env/secrets-prod.env
GHCR_REGISTRY=ghcr.io
GHCR_ORG=ales-imt

# Version des images, passée par le makefile (IMAGE_TAG). C'est ce tag, et non
# `latest`, qui est déployé : une mise en production est un changement de
# version tracé, pas un redémarrage qui ramasse ce qui traîne.
IMAGE_TAG="${IMAGE_TAG:-dev}"

# UID de service dans l'image export-webdfd : celui de userdfx sur la VM.
EXPORT_UID="${EXPORT_UID:-1001}"
GITHUB_DOCKER_TOKEN=$(grep '^GITHUB_DOCKER_TOKEN=' "$SECRETS_FILE" | cut -d'=' -f2)

echo "--- ⚙️  Génération sqlc ---"
(cd backend/common  && sqlc generate)
(cd backend/admin   && sqlc generate)
(cd backend/student && sqlc generate)
(cd backend/sync    && sqlc generate)

./infras/run/build-admin.sh prod "$IMAGE_TAG"
./infras/run/build-eleve.sh prod "$IMAGE_TAG"
./infras/run/build-sync.sh prod "$IMAGE_TAG"
./infras/run/build-export-webdfd.sh prod "$IMAGE_TAG" "$EXPORT_UID"

echo "--- 🔑 Connexion à GHCR ---"
echo "$GITHUB_DOCKER_TOKEN" | docker login "$GHCR_REGISTRY" -u "$GHCR_ORG" --password-stdin

echo "--- 🏷️ Tag des images ($IMAGE_TAG) ---"
for img in rex-admin rex-eleve rex-sync rex-export-webdfd; do
    docker tag "$img:$IMAGE_TAG" "$GHCR_REGISTRY/$GHCR_ORG/$img:$IMAGE_TAG"
done

echo "--- 🚀 Push vers GHCR ---"
for img in rex-admin rex-eleve rex-sync rex-export-webdfd; do
    docker push "$GHCR_REGISTRY/$GHCR_ORG/$img:$IMAGE_TAG"
done

echo "--- ✅ Images publiées sur $GHCR_REGISTRY/$GHCR_ORG en version $IMAGE_TAG ---"
