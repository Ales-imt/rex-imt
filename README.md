## Licence
Ce projet est distribué sous licence [AGPL v3](LICENSE).


Coffre fort numerique:
    sudo apt  install age
    age-keygen -o cle_privee.txt
La clef public est dans cle_privee.txt

Protéger la clé privée par passphrase
age-keygen | age -p > cle_privee_chiffree.age

Decrypter une donnee chiffree
age -d cle_privee_chiffree.age | age -d -i - donnees_a_dechiffrer.age

age --decrypt -i cle_privee.txt t.txt


Pour serveur-ia, generation de clef:
tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 48 ; echo ''

Pour utiliser cybema en local:
    dans /etc/hosts
   127.0.0.1       webdfd.mines-ales.fr
   sudo systemctl stop apache2
   ssh -L 5433:sql2.mines-ales.fr:5433 -L 3306:sql2.mines-ales.fr:3306 -L 80:webdfd.mines-ales.fr:80 userdfx@vecu-etudiant-eleves-2.mines-ales.fr


Pour expo,
   npx expo start

dump de la bd postgresql:
ssh -L 5433:sql2.mines-ales.fr:5433 userde@vecu-etudiant.mines-ales.fr
pg_dump -h localhost -p 5433  -U devedbuserext  -d devedb -F c -f db_rex_backup.dump


dump de mysql

source .vscode/secrets-prod.env && docker run --rm --network host mariadb:latest   mariadb-dump -h $MARIADB_HOST -P $MARIADB_PORT -u $MARIADB_USER -p$MARIADB_PASSWORD   --skip-ssl --skip-lock-tables  "$MARIADB_DB" > dump_cyber_notes_$(date +%Y%m%d).sql

