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
   sudo ssh -L 80:webdfd.mines-ales.fr:80 userdfx@vecu-etudiant-eleves-2.mines-ales.fr


Pour expo,
   npx expo start

