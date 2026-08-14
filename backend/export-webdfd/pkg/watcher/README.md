# rex-export-webdfd — sauvegarde HFSQL → JSON

Surveille le répertoire où sont déposées les sauvegardes de cybema (partage
Samba) et convertit chaque archive zip en JSON. Linux uniquement : inotify est
appelé directement via `golang.org/x/sys/unix`.

Le service ne connaît **ni base de données ni réseau**. Il lit un répertoire,
écrit un autre répertoire, et rien d'autre. Cette pauvreté est délibérée : il
donne à mâcher des fichiers venus d'un partage à un exécutable WinDev sous wine,
la pire surface d'attaque de l'ensemble, et n'a donc aucune raison de détenir un
identifiant PostgreSQL.

Sa sortie est reprise par `rex-sync` selon le contrat décrit dans
[`back-rex-common/pkg/hfsqlexport`](../../../common/pkg/hfsqlexport).

## Les deux modes d'exécution

C'est le seul choix structurant de ce service, réglé par `migration.watcher.runner`.

| Mode | Où tourne le service | Comment l'export s'exécute |
| --- | --- | --- |
| `docker` (défaut) | sur l'hôte | `docker run wine32-hf55` |
| `local` | **dans** l'image wine32-hf55 | wine appelé directement |

Le mode `docker` ne se conteneurise pas. Les chemins des `-v` sont résolus par
le démon, pas par l'appelant : depuis un conteneur il faudrait lui monter le
socket du démon — donc lui accorder l'équivalent du root sur l'hôte — et monter
`data` au même chemin des deux côtés, faute de quoi le démon crée silencieusement
un répertoire vide et l'export sort sans données. En prod, sous podman, ce socket
n'existe pas.

Le mode `local` est la réponse : au lieu de lancer le conteneur wine, **on est**
le conteneur wine. L'image de ce service se construit `FROM wine32-hf55`, on y
ajoute ce binaire, et l'export devient un appel local.

### Contrat avec l'image wine32-hf55

En mode `local`, le script d'export reçoit ses chemins par l'environnement :

| Variable | Rôle |
| --- | --- |
| `HFSQL_BASE` | base HFSQL dézippée, à monter en `D:` |
| `HFSQL_OUT` | répertoire de sortie, à monter en `E:` |

L'`entrypoint.sh` actuel de `wine32-hf55` code ces chemins en dur (`/data` et
`/out`), car ils venaient des `-v`. **Il doit être adapté pour lire ces deux
variables** avant que le mode `local` puisse servir. C'est la seule modification
requise côté image.

## Configuration

Voir [`cmd/config.yaml`](../../cmd/config.yaml).

| Clé | Défaut | Rôle |
| --- | --- | --- |
| `depot` | — | **obligatoire**, chemin absolu : répertoire surveillé |
| `data` | — | **obligatoire**, chemin absolu : bases dézippées et exports |
| `runner` | `docker` | `docker` ou `local` |
| `image` | `wine32-hf55` | mode docker : image de l'export |
| `entree` | `/entrypoint.sh` | mode local : script d'export |
| `timeout` | `15m` | durée maximale d'un export |
| `keep` | `0` (tous) | exports conservés sous `data` |

La configuration est validée au démarrage : un chemin relatif ou manquant, un
mode inconnu, et le service s'arrête — cela ne se découvre pas au premier dépôt,
la nuit où tombe la sauvegarde.

## Traitement d'un dépôt

1. **Vérification** : l'archive est ouverte et le CRC de toutes ses entrées est
   contrôlé (équivalent de `unzip -t`). Un fichier qui n'est pas un zip valide
   est ignoré avec un message, sans arrêter le service.
2. **Dézippage** dans `<data>/` : le répertoire racine de l'archive (ex.
   `20250707.bd`) est purgé puis extrait. Les entrées sortant du répertoire
   cible (« zip slip ») sont rejetées.
3. **Écriture** de `emptemp.WDD` dans la base extraite. L'analyse WinDev est
   embarquée dans le binaire (`go:embed`) : elle décrit la structure des
   fichiers HFSQL, donc elle suit le code et non les données.
4. **Sortie** : création de `<data>/export-AAAAMMJJ-HHMMSS/json`, un répertoire
   par export pour éviter les conflits entre dépôts rapprochés.
5. **Export** : selon le mode, `docker run` ou wine en direct.
6. **Marqueur** : `<data>/export-AAAAMMJJ-HHMMSS/.termine`, écrit **en dernier**.
7. **Purge** des exports au-delà de `keep`.

L'ordre des étapes 6 et 7 n'est pas cosmétique. Un export dure plusieurs minutes
et son répertoire existe dès la première seconde : sans le marqueur, `rex-sync`
lirait tôt ou tard un instantané à moitié écrit, donc un planning amputé — et
annulerait des séances à tort.

Les dépôts sont traités en série : deux exports simultanés se disputeraient les
mêmes `.FIC`.

## Notes inotify

- Le masque est `IN_CLOSE_WRITE | IN_MOVED_TO`. `IN_CLOSE_WRITE` garantit que la
  copie est terminée : contrairement à `fsnotify`, qui n'expose pas cet
  événement, on ne se réveille jamais sur une archive partielle.
- Un dépôt déplacé depuis un **autre** système de fichiers n'émet pas
  `IN_MOVED_TO` mais un create + write, donc un `IN_CLOSE_WRITE` : les deux cas
  sont couverts.
- inotify n'est pas récursif : seul le répertoire de dépôt est surveillé.
- Un débordement de la file du noyau (`IN_Q_OVERFLOW`, réglable via
  `/proc/sys/fs/inotify/max_queued_events`) est signalé comme une erreur fatale
  plutôt que silencieusement ignoré — des dépôts auraient été perdus.
- Le descripteur est ouvert en `IN_NONBLOCK` et confié à un `*os.File` : le
  runtime Go l'enregistre dans son poller, ce qui permet à `Close()` de
  débloquer le `Read()` en cours à l'annulation du contexte.

Le partage étant servi par Samba **sur cette machine**, les écritures passent par
le VFS local et les événements arrivent normalement.

Ici l'écoute se justifie : rater un dépôt n'a pas de conséquence durable, la
sauvegarde suivante sera prise. `rex-sync`, lui, sonde — rater un export le
laisserait complet mais ignoré jusqu'au dépôt suivant (voir
[`sync/README.md`](../../../sync/README.md)).

## Tests

```sh
go test ./...
```

Ils couvrent le décodage inotify (`IN_CLOSE_WRITE`, `IN_MOVED_TO`, arrêt propre),
la validation de la configuration et son décodage YAML, l'intégrité de l'analyse
embarquée et la manipulation des archives (racine unique, archive tronquée, zip
slip, extraction). Ils n'appellent ni docker ni wine.
