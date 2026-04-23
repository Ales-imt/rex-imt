#!/bin/bash

# --- PARAMÈTRES DE BASE ---
# Nom du répertoire qui contiendra tous les fichiers (évite la pollution)
DIR_NAME="mtls_certs"
# Nom de domaine/CN pour le serveur (doit correspondre à ServerName dans la config TLS)
SERVER_HOST="localhost"
# Nom pour le client (CN)
CLIENT_NAME="mtls-client"

# Durée de validité des certificats (en jours)
VALIDITY_DAYS=365

echo "--- Préparation de l'environnement ---"
rm -rf "$DIR_NAME"
mkdir -p "$DIR_NAME"
cd "$DIR_NAME" || exit

# --- 1. CRÉATION DE L'AUTORITÉ DE CERTIFICATION (CA) ---

echo "--- 1. Création de la CA Racine ---"
# Générer la clé privée de la CA (non chiffrée, pour la simplicité du test)
openssl genrsa -out ca.key 2048

# Générer le certificat racine auto-signé
# -x509 : Demande un certificat auto-signé
# -days : Durée de validité
# -subj : Informations sur le sujet (CN=Common Name)
openssl req -new -x509 -days $VALIDITY_DAYS -key ca.key -out ca.crt \
    -subj "/C=FR/ST=IDF/O=Test CA/CN=My Test Root CA"

echo "CA créée : ca.key (clé), ca.crt (certificat)"

# --- 2. CRÉATION DES CLEFS ET CERTIFICATS DU SERVEUR ---

echo "--- 2. Création des clés et certificats du Serveur ---"

# 2a. Générer la clé privée du serveur
openssl genrsa -out server.key 2048

# 2b. Créer le fichier de configuration OpenSSL pour les SANs (Subject Alternative Names)
# Le SAN est essentiel pour éviter les avertissements de domaine non correspondant dans les navigateurs.
cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = $SERVER_HOST
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

# 2c. Générer la demande de signature de certificat (CSR) du serveur
openssl req -new -key server.key -out server.csr \
    -subj "/C=FR/ST=IDF/O=Test Server/CN=$SERVER_HOST" \
    -config <(printf "[req]\ndistinguished_name=req\n[req_distinguished_name]")

# 2d. Signer le CSR du serveur avec la clé de la CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days $VALIDITY_DAYS -sha256 \
    -extfile server.ext

echo "Serveur créé : server.key (clé), server.crt (certificat)"

# --- 3. CRÉATION DES CLEFS ET CERTIFICATS DU CLIENT ---

echo "--- 3. Création des clés et certificats du Client ---"

# 3a. Générer la clé privée du client (non chiffrée)
openssl genrsa -out client.key 2048

# 3b. Créer le fichier de configuration OpenSSL pour les usages étendus (Extended Key Usage)
cat > client.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth # Clé essentielle pour le mTLS
EOF

# 3c. Générer la demande de signature de certificat (CSR) du client
openssl req -new -key client.key -out client.csr \
    -subj "/C=FR/ST=IDF/O=Test Client/CN=$CLIENT_NAME" \
    -config <(printf "[req]\ndistinguished_name=req\n[req_distinguished_name]")

# 3d. Signer le CSR du client avec la clé de la CA
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -days $VALIDITY_DAYS -sha256 \
    -extfile client.ext

echo "Client créé : client.key (clé), client.crt (certificat)"

echo ""
echo "--- Installation des fichiers ---"
echo "Les fichiers sont dans le répertoire $DIR_NAME"
echo ""
echo "POUR L'APPLICATION GO CLIENT:"
echo "  Client Cert: $DIR_NAME/client.crt"
echo "  Client Key:  $DIR_NAME/client.key"
echo "  Server CA:   $DIR_NAME/ca.crt (Utilisé pour RootCAs)"
echo ""
echo "POUR LE SERVEUR APACHE/GO (mTLS):"
echo "  Server Cert: $DIR_NAME/server.crt"
echo "  Server Key:  $DIR_NAME/server.key"
echo "  Client CA:   $DIR_NAME/ca.crt (Utilisé pour SSLCACertificateFile/ClientCAs)"

cd ..
