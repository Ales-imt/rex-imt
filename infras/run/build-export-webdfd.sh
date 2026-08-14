#!/bin/sh
set -e

# UID/GID de l'utilisateur de service DANS le conteneur. Il doit correspondre au
# propriétaire des répertoires depot/data sur l'hôte : 1000 en local (l'UID du
# développeur), 1001 en prod (celui de userdfx sur la VM).
#
# En prod, l'unité Quadlet doit en plus porter `UserNS=keep-id` : podman
# rootless remappe sinon cet UID vers une plage subuid, et les exports produits
# n'appartiendraient pas à userdfx.
#
# BUILD_UID/BUILD_GID reçoivent la MÊME valeur : wine exige que le WINEPREFIX
# lui appartienne, un préfixe construit sous un autre UID est refusé quels que
# soient ses droits. Voir le commentaire du stage wine-base.
ENV="${1:-prod}"
IMAGE_TAG="${2:-dev}"
if [ "$ENV" = "prod" ]; then
    SERVICE_UID="${3:-1001}"
else
    SERVICE_UID="${3:-1000}"
fi
SERVICE_GID="${4:-$SERVICE_UID}"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "--- 🐳 Build de l'image export-webdfd (target: $ENV, tag: $IMAGE_TAG, uid: $SERVICE_UID) ---"
docker build -f ./infras/run/build/Dockerfile.export-webdfd \
    --target "$ENV" \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    --build-arg UID="$SERVICE_UID" \
    --build-arg GID="$SERVICE_GID" \
    --build-arg BUILD_UID="$SERVICE_UID" \
    --build-arg BUILD_GID="$SERVICE_GID" \
    -t "rex-export-webdfd:$IMAGE_TAG" \
    -t rex-export-webdfd .
