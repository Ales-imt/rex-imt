#!/bin/sh
set -e

nginx

exec dlv exec /usr/local/bin/rex-eleve-server \
    --listen=:2346 --headless --api-version=2 --accept-multiclient --continue \
    -- /opt/rex-eleve/conf/config.yaml
