liquibase update
liquibase rollbackCount 1

liquibase updateToTag V0_1
liquibase rollback V0_1

donne le sql pour aller a la V2
liquibase updateSQL --toTag=V0_2