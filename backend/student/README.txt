Pour la generation du certificat

openssl req -x509 -newkey rsa:4096 -keyout cle.key -out certificat.crt -days 365 -nodes

openssl req -x509 -newkey rsa:2048 -sha256 -days 365 -nodes -keyout server.key -out server.crt   -subj "/CN=10.202.99.86"

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout server.key -out server.crt \
  -config openssl.cnf

pour importer les services de commun:
go mod edit -replace back-rex-common=../back-rex-common
go mod tidy


modif a reporter en BD:

ALTER TABLE public.user DROP COLUMN ldapid;
ALTER TABLE refresh_tokens ADD COLUMN session TEXT;
ALTER TABLE refresh_tokens ADD COLUMN token_version INT;

ALTER TABLE feedback ADD COLUMN remote_addr TEXT;
