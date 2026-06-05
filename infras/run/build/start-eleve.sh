#!/bin/sh
set -e

nginx

exec /usr/local/bin/rex-eleve-server /opt/rex-eleve/conf/config.yaml
