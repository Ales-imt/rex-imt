#!/bin/sh
set -e

ENV="${1:-prod}"
IMAGE_TAG="${2:-dev}"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "--- 🐳 Build de l'image élève (target: $ENV, tag: $IMAGE_TAG) ---"
docker build -f ./infras/run/build/Dockerfile.eleve \
    --target "$ENV" \
    --build-arg VERSION="$VERSION" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t "rex-eleve:$IMAGE_TAG" \
    -t rex-eleve .
