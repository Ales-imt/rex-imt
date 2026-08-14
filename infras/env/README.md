# infras/env — configuration par environnement

Ces fichiers alimentent les `${VAR}` des `backend/*/cmd/config.yaml`, les
conteneurs (`--env-file`), Ansible (via `generate-vault-vars.sh`), Liquibase et
la création des bases.

Ils vivaient dans `.vscode/`. Ce n'était pas leur place : un répertoire
d'éditeur se supprime sans y penser, se trouve souvent dans un `.gitignore`
global, et n'a aucun rapport avec le déploiement d'une production.

## La règle de partage

Deux fichiers par environnement, sur un critère unique.

| | Contenu | Versionné |
|---|---|---|
| `secrets-{local,prod}.env` | ce dont la fuite est un incident et qui **se révoque** : mots de passe, jetons, clés de signature | **non** |
| `config-local.env` | la topologie de développement : adresses, ports, noms de bases, comptes, clés publiques | **oui** |
| `config-prod.env` | la même chose pour la production | **oui** |
| `secrets.env.example` | la **liste** des secrets attendus, valeurs vides | **oui** |

Un identifiant n'est pas une preuve d'identité : `POSTGRES_USER` est de la
topologie, `POSTGRES_PASSWORD` est un secret. De même, `AGE_PUBLIC_KEY` est une
clé *publique* — elle est faite pour circuler.

Les deux `config-*.env` sont versionnés à dessein. Le local décrit le réseau
docker du dépôt (voir `infras/container/compose.yaml`) et permet de démarrer
sans rien deviner ; celui de production met sous revue les changements de
serveur ou de chemin. Le dépôt versionne de toute façon déjà la topologie de
prod — inventaire ansible, `group_vars`, chemins `/srv` — et l'en soustraire
n'aurait protégé personne.

Une exception vit dans les fichiers de secrets sans en être un :
`WITNESS_RECIPIENT`, destinataire du témoin d'ancrage. Ce n'est pas un secret
mais une donnée à caractère personnel, qui n'a pas à entrer dans l'historique
du dépôt (cf. `docs/rgpd-dpo.md`).

## Premier démarrage

```sh
cp infras/env/secrets.env.example infras/env/secrets-local.env
# puis renseigner les valeurs
```

Pour la production, il faut en plus `secrets-prod.env` et `config-prod.env`.

## Le piège que ces fichiers tendent

`os.ExpandEnv` remplace une variable absente par une **chaîne vide**, sans le
moindre message. Un secret oublié ne se voit donc pas au démarrage : il se voit
le jour où l'authentification échoue — ou jamais, si la fonctionnalité se
désactive en silence. C'est le cas de `presence.witness`, que
`witnessEnabled()` désactive sans un mot si `smtp.host` arrive vide.

Toutes les variables vides ne sont pas des oublis : `SMTP_USERNAME` et
`SMTP_PASSWORD` le sont légitimement, le relais interne n'authentifiant pas
(`pkg/mailer/mailer.go` ne tente l'authentification que si un identifiant est
fourni). `AUREGA_*` le sont aussi, cette source n'étant pas encore en service.

Deux garde-fous en découlent :

- `secrets.env.example` est versionné, pour que la liste attendue soit
  vérifiable ;
- les scripts s'arrêtent si un fichier manque, plutôt que de continuer avec des
  valeurs vides.

Aucun des deux ne détecte une variable *présente mais vide* : cela reste à la
charge du lecteur.

## Le lien avec Ansible

`generate-vault-vars.sh` lit ces deux fichiers et en produit `vault-vars.yml`,
chargé par `--extra-vars`. Y transitent les points de terminaison, les secrets,
et les chemins des sauvegardes HFSQL (`hfsql_depot`, `hfsql_data`) — une seule
place par environnement pour cette valeur, ici comme en local.

**`--extra-vars` est la précédence la plus haute d'Ansible**, au-dessus de
`vars_files`. Une clé transmise par le vault ne doit donc pas être redéfinie
dans `infras/ansible/vars/` : elle y serait inerte, et divergerait en silence.

Le reste — `app_dir`, `container_name`, `container_port`, `container_volumes` —
décrit la disposition du déploiement et vit dans `vars/`, versionné.
