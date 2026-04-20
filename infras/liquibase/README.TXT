liquibase update
liquibase rollbackCount 1

liquibase updateToTag V0_1
liquibase rollback V0_1

donne le sql pour aller a la V2
liquibase updateSQL --toTag=V0_2

docker cp ~/devedb_backup.dump postgres-16.10-alpine-rex:/tmp/devedb_backup.dump
docker exec -i postgres-16.10-alpine-rex  pg_restore --no-owner -U postgres -d db_rex /tmp/devedb_backup.dump
cd infras/liquibase && liquibase changelogSyncToTag v0.0.0


