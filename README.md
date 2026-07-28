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


Pour utiliser cybema en local, necessite proxy socks5h. 
    ssh -N -D 0.0.0.0:1080   -L 5433:sql2.mines-ales.fr:5433   -L 3306:sql2.mines-ales.fr:3306   -L 0.0.0.0:2525:mail.mines-ales.fr:25  userdfx@vecu-etudiant-eleves-2.mines-ales.fr

    curl --proxy socks5h://127.0.0.1:1080 "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=planning_txt&DATEDEBUT=20260223&DATEFIN=20260301&TYPECLE=evcleunik&VALCLE=17979"
Les containers en local doive avoir :
    -e HTTP_PROXY=socks5h://host.docker.internal:1080 \
    -e NO_PROXY=10.20.1.4,10.20.1.5,10.20.1.6,localhost,127.0.0.1 \

Pour utiliser le telephone avec un container et eviter https pour la camera:
  /home/vjo/Android/Sdk/platform-tools/adb reverse tcp:8131 tcp:8131
  socat TCP-LISTEN:8131,bind=localhost,fork TCP:10.20.1.11:8131


Pour expo,
   npx expo start

dump de la bd postgresql:
ssh -L 5433:sql2.mines-ales.fr:5433 userde@vecu-etudiant.mines-ales.fr
pg_dump -h localhost -p 5433  -U devedbuserext  -d devedb -F c -f db_rex_backup.dump


dump de mysql

source .vscode/secrets-prod.env && docker run --rm --network host mariadb:latest   mariadb-dump -h $MARIADB_HOST -P $MARIADB_PORT -u $MARIADB_USER -p$MARIADB_PASSWORD   --skip-ssl --skip-lock-tables  "$MARIADB_DB" > dump_cyber_notes_$(date +%Y%m%d).sql

