# LDAP local de développement

Serveur OpenLDAP (`osixia/openldap:1.5.0`) préchargé avec des utilisateurs de test pour développer sans accès à `ldap.mines-ales.fr`.

## Démarrage

```bash
make ldap-start
# ou directement :
cd infras/container && docker compose up -d openldap
```

Le serveur écoute sur `ldap://localhost:3890` avec le BaseDN `dc=ema,dc=fr`.

## Utilisateurs de test

| uid | email | mot de passe | promotion | rôle |
|-----|-------|-------------|-----------|------|
| alice.martin | alice.martin@etu.mines-ales.fr | password | 3A-INFO | étudiant |
| bob.dupont | bob.dupont@etu.mines-ales.fr | password | 2A-GC | étudiant |
| clara.leroy | clara.leroy@etu.mines-ales.fr | password | 1A | étudiant |
| jean.bernard | jean.bernard@mines-ales.fr | password | PROF | enseignant/admin |
| david.simon | david.simon@etu.mines-ales.fr | password | 2A-INFO-RED | étudiant redoublant |

## Tester la connexion depuis le terminal

Recherche par email :

```bash
ldapsearch -x -H ldap://localhost:3890 \
  -D "cn=admin,dc=ema,dc=fr" \
  -w adminpassword \
  -b "dc=ema,dc=fr" \
  "(mail=alice.martin@etu.mines-ales.fr)"
```

Lister tous les utilisateurs :

```bash
ldapsearch -x -H ldap://localhost:3890 \
  -D "cn=admin,dc=ema,dc=fr" \
  -w adminpassword \
  -b "ou=people,dc=ema,dc=fr" \
  "(objectClass=inetOrgPerson)"
```

## Réinitialiser les données

Les fichiers bootstrap ne sont chargés qu'au **premier démarrage** (volumes vides).
Pour repartir de zéro :

```bash
make ldap-reset
make ldap-start
```

## Notes

- Les fichiers `.ldif` dans `bootstrap/` sont injectés uniquement quand les volumes `openldap-data` et `openldap-config` sont vides (premier démarrage). Le répertoire n'est pas monté en `:ro` car `osixia/openldap` doit pouvoir faire un `chown` dessus au démarrage.
- `LDAP_TLS: "false"` est requis pour que le backend Go se connecte en `ldap://` sans TLS.
- **Ne jamais utiliser ces credentials en production.**
