#!/bin/sh
set -e

nginx

exec /usr/local/bin/rex-admin-server /opt/rex-admin/conf/config.yaml
