#!/bin/sh
set -e

crond -b
nginx

exec dlv exec /usr/local/bin/rex-admin-server \
    --listen=:2345 --headless --api-version=2 --accept-multiclient --continue \
    -- /opt/rex-admin/conf/config.yaml
