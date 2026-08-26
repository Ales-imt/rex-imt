#!/bin/sh
set -e

ENV="${1:-prod}"
IMAGE_TAG="${2:-dev}"
# --match limite aux tags de version (v1.2.3) : un tag "backup/*" plus proche
# de HEAD rendrait sinon la description illisible. Sans tag de version,
# --always retombe sur le hash court, qui reflète quand même exactement le
# commit construit.
VERSION=$(git describe --tags --always --dirty --match 'v[0-9]*' 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "--- 🐳 Build de l'image admin (target: $ENV, tag: $IMAGE_TAG) ---"
docker build -f ./infras/run/build/Dockerfile.admin \
    --target "$ENV" \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t "rex-admin:$IMAGE_TAG" \
    -t rex-admin .
