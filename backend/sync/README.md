# rex-sync — référentiel scolaire → PostgreSQL

Maintient dans PostgreSQL les promotions, professeurs, élèves, matières,
groupes et séances, à partir de la source amont désignée par la configuration.
Il ne sert aucune donnée : seuls `/health` et `/version` sont exposés.

## Découpage

```
pkg/source/            CE QUE dit l'amont
  source.go              interface Source + DTO (Promo, Personne, Creneau)
  webdfd/                cybema en HTTP (cgiempt.exe, Windows-1252)
  hfsql/                 exports JSON de rex-export-webdfd
  connect/               fabrique : configuration → source
pkg/migration/         CE QU'ON EN FAIT
  sync.go                les six étapes, dans l'ordre
  promotion.go profs.go eleves.go planning.go listegroupe.go
  planning_annulation.go query.sql(.go)
pkg/scheduler/         QUAND on le fait
```

Ces trois axes sont séparés parce qu'ils changent pour des raisons différentes.
**C'est la seule chose qui compte pour le remplacement de webdfd par auréga** :
il suffira d'ajouter `pkg/source/aurega/` et une entrée dans `connect`. Le
moteur, qui porte toute la logique métier — résolution des identités, périodes,
annulation des séances disparues — ne bougera pas.

La fabrique vit dans un sous-paquet à cause d'un cycle d'import : les sources
concrètes importent `pkg/source` pour son interface, qui ne peut donc pas les
importer en retour. Même découpage que `back-rex-eleve/pkg/programme/connect`.

## Les deux sources

| | `webdfd` | `hfsql` |
| --- | --- | --- |
| Origine | cybema interrogé en HTTP | exports JSON de `rex-export-webdfd` |
| Déclenchement | toutes les 2 h, réessai 5 min | à chaque export complet |
| Fraîcheur | temps réel | celle de la dernière sauvegarde |
| Encodage | Windows-1252 | UTF-8, chaînes à largeur fixe |
| Libellés du planning | fournis par le flux | résolus par jointure |

Les deux décrivent **le même référentiel** et portent les mêmes identifiants —
les clés primaires HFSQL (`P0CLEUNIK` = `P0`, `EVCLEUNIK` = `EV`,
`PLCLEUNIK` = `PL`). C'est pourquoi les deux écrivent `source = 'webdfd'` dans
les tables `migration.*_map` : cette colonne désigne le référentiel amont, pas le
transport. Les distinguer créerait un doublon de chaque utilisateur, promotion et
séance au premier basculement.

Une source absente ou inconnue **arrête le démarrage**. Se rabattre sur l'autre
synchroniserait un référentiel différent de celui attendu, sans que personne ne
s'en aperçoive avant qu'un cours ne manque à l'appel.

## Pourquoi un sondage, et non inotify

En source `hfsql`, le service cherche toutes les minutes un export plus récent
que le dernier traité, sous `migration.hfsql.data` monté en lecture seule.

Un marqueur suffirait à signaler la fin d'un export, et inotify fonctionnerait
dans le déploiement visé — deux conteneurs sur le même hôte, bind mounts du même
répertoire. Ce n'est donc pas une question de portabilité.

La raison est qu'inotify est déclenché sur **front**, le sondage sur **niveau**.
Un événement perdu — file du noyau saturée, service redémarré pendant que
l'export se termine — et la synchronisation n'a jamais lieu : le fichier est là,
complet, mais plus personne ne viendra le regarder avant le dépôt suivant. Le
sondage ne demande jamais « que s'est-il passé ? » mais « quel est le dernier
export complet ? », et rattrape donc tout seul.

Le prix est une latence d'au plus une minute, sur un événement qui survient une
fois par jour. Si elle devenait gênante, la bonne réponse serait d'ajouter
inotify comme sonnette **sans retirer** le sondage, gardé comme filet.

Seuls les exports portant leur marqueur `.termine` sont retenus — voir
[`back-rex-common/pkg/hfsqlexport`](../common/pkg/hfsqlexport). Aucun état n'est
persisté : au redémarrage le dernier export est resynchronisé, ce qui ne coûte
qu'un cycle puisque toutes les étapes sont en upsert.

## Configuration

Voir [`cmd/config.yaml`](cmd/config.yaml). Le service a besoin de `database`,
`server.port`, `migration.source`, puis de `webdfd` **ou** de `migration.hfsql`
selon la source.

## Génération sqlc

`db.go`, `models.go` et `query.sql.go` sont générés et gitignorés :

```sh
cd backend/sync && sqlc generate
```

C'est déjà câblé dans `infras/db-to-code.sh` et `infras/run/push-container.sh`.

## Tests

```sh
go test ./...
```

`TestSyncPromotions` écrit dans une vraie base et se saute si elle est
injoignable. Le reste — parsing webdfd, lecture des exports, périodes, année de
rattachement, annulation des séances — ne dépend de rien.
