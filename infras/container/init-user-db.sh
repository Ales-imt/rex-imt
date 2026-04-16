#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL

	CREATE USER db_rex WITH PASSWORD 'db_rex';
	CREATE DATABASE db_rex WITH OWNER = db_rex;
	GRANT ALL PRIVILEGES ON DATABASE db_rex TO db_rex;
	
EOSQL

