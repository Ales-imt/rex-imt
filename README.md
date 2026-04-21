Coffre fort numerique:
    sudo apt  install age
    age-keygen -o cle_privee.txt
La clef public est dans cle_privee.txt

Protéger la clé privée par passphrase
age-keygen | age -p > cle_privee_chiffree.age

Decrypter une donnee chiffree
age -d cle_privee_chiffree.age | age -d -i - donnees_a_dechiffrer.age

age --decrypt -i cle_privee.txt t.txt

