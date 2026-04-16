#!/bin/bash

# Récupère l'heure de compilation au format RFC3339
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Récupère le hash du dernier commit Git
VERSION=$(git rev-parse HEAD)

# Compile le programme en injectant les variables
go build -ldflags "-X 'main.buildTime=${BUILD_TIME}' -X 'main.version=${VERSION}'" -o cmd_admin


