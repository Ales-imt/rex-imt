#!/bin/sh
set -e

crond -b

nginx

exec /usr/local/bin/rex-admin-server /opt/rex-admin/conf/config.yaml
