# REX-IMT — Compilation et déploiement

## Présentation

REX-IMT est une plateforme d'évaluation composée de trois applications :

- **Admin** : interface de gestion à destination des enseignants (backend Go + frontend React)
- **Élève web** : interface web à destination des étudiants (backend Go + frontend Expo/web)
- **Élève mobile** : application mobile multiplateforme pour les étudiants (React Native / Expo)

Les applications admin et élève web sont packagées dans des images Docker publiées sur le registre GitHub Container Registry (GHCR) et déployées sur des VMs dédiées via Ansible.

---

## Structure du projet

```
rex-imt/
├── apps/
│   ├── web-admin/          # Frontend React (admin)
│   └── mobile/             # Application React Native / Expo (élève)
├── backend/
│   ├── common/             # Module Go partagé (admin + élève)
│   ├── admin/              # Backend Go — serveur admin
│   └── student/            # Backend Go — serveur élève
├── infras/
│   ├── container/          # Docker Compose (environnement local : PostgreSQL, LDAP, Ollama)
│   ├── run/                # Dockerfiles, scripts de build/lancement, configs nginx
│   ├── liquibase/          # Migrations de base de données
│   ├── migration/          # Outil Go de migration des données
│   └── ansible/            # Déploiement sur les VMs prod
├── makefile                # Point d'entrée principal
├── makefile.local          # Cibles local
└── makefile.prod           # Cibles prod
```

---

## Prérequis

- Docker
- Go 1.25+
- Node.js 22+
- `sqlc` (génération de code depuis le schéma SQL)
- `pg_dump` (extraction du schéma PostgreSQL)
- Ansible (déploiement prod)
- `age` (chiffrement des secrets — voir section coffre numérique)

---

## Initialisation des dépendances Node.js

Avant de builder les conteneurs Docker ou de lancer les applications en développement, il faut que le fichier `package-lock.json` existe dans chaque application Node.js.

### Pourquoi c'est obligatoire

Les Dockerfiles utilisent `npm ci` pour installer les dépendances à l'intérieur du conteneur. `npm ci` exige la présence d'un `package-lock.json` — sans lui, le build échoue immédiatement.

Le `package-lock.json` n'est **pas généré pendant le build Docker** : Docker copie le fichier depuis la machine locale (`COPY apps/.../package*.json ./`), puis `npm ci` l'utilise. Il doit donc exister localement avant de lancer `docker build`.

### Conséquence pour les builds reproductibles

Sans `package-lock.json` commité dans git, chaque développeur qui clone le dépôt et lance `docker build` obtiendra des versions de dépendances potentiellement différentes (selon ce que npm résout au moment du build). Cela peut provoquer des comportements différents entre les environnements.

Avec `package-lock.json` commité, tous les développeurs installent exactement les mêmes versions — le build est déterministe.

### Génération (à faire une seule fois par application)

```bash
# Frontend admin
cd apps/web-admin
npm install --package-lock-only   # génère package-lock.json sans installer node_modules

# Application mobile
cd apps/mobile
npm install --package-lock-only
```

Commiter ensuite les fichiers générés :

```bash
git add apps/web-admin/package-lock.json apps/mobile/package-lock.json
git commit -m "chore: add package-lock.json for reproducible builds"
```

> À refaire uniquement lors de l'ajout, la suppression ou la mise à jour d'une dépendance dans `package.json`.

---

## Workflow local

La commande principale pour lancer un environnement de développement complet est :

```bash
make local
```

Elle enchaîne dans l'ordre :
1. `infra-local` — démarre les conteneurs Docker locaux (PostgreSQL, LDAP, Ollama)
2. `migration-local` — applique la migration v0 → v1 (script `migrate-v0-to-v1.sh`)
3. `db-to-code-local` — extrait le schéma et génère le code sqlc
4. `start-local` — build et lance les conteneurs admin et élève

### Gestion individuelle des conteneurs

```bash
make start-admin    # build (target local) + lance le container admin sur :8121 (debug :2345)
make start-eleve    # build (target local) + lance le container élève sur :8131 (debug :2346)
make stop-admin     # arrête le container admin
make stop-eleve     # arrête le container élève
make stop-local     # arrête admin + élève
```

Les variables de connexion sont lues depuis `.vscode/secrets-local.env`.

### Nettoyage

```bash
make clean    # arrête les conteneurs Docker Compose et supprime les libs Liquibase
```

---

## Infra locale (Docker Compose)

Le fichier `infras/container/compose.yaml` démarre les services suivants, tous sur le réseau bridge interne `rex-net` (`10.20.1.0/24`) :

| Service | Image | IP | Port | Rôle |
|---|---|---|---|---|
| `db` | `postgres:16.10-alpine` | `10.20.1.4` | 5432 | Base de données PostgreSQL |
| `openldap` | `osixia/openldap:1.5.0` | `10.20.1.5` | 389 | Annuaire LDAP (auth) |
| `mariadb` | `mariadb:latest` | `10.20.1.6` | 3306 | Base de données MariaDB |
| `ollama` | `ollama/ollama:latest` | `10.20.1.2` | 11434 | Serveur LLM local |

Les containers admin et élève rejoignent ce même réseau au lancement :

| Conteneur | IP | Port applicatif | Port debug (Delve) |
|---|---|---|---|
| `rex-admin` | `10.20.1.10` | 8121 | 2345 |
| `rex-eleve` | `10.20.1.11` | 8131 | 2346 |

Aucun port n'est mappé sur l'hôte — l'accès se fait directement via l'IP du bridge (ex. `http://10.20.1.10:8121`). Le réseau `10.20.1.0/24` est en dehors du pool d'allocation automatique de Docker (`172.17.0.0/12`), ce qui évite les conflits entre projets.

Au démarrage, le service `ollama-init` tire automatiquement le modèle `mistral-nemo`.

> Le support GPU NVIDIA pour Ollama est disponible mais désactivé par défaut (décommenter la section `deploy.resources` dans le compose).

---

## Génération du code (db-to-code)

Le script `infras/db-to-code.sh` extrait les schémas des deux bases de données et régénère le code `sqlc` :

```bash
make db-to-code-local   # utilise secrets-local.env
```

Étapes exécutées :
1. `pg_dump` — extrait le schéma PostgreSQL vers `backend/schema.sql`
2. `mariadb-dump` (via Docker) — extrait le schéma MariaDB vers `backend/schema_maria_db.sql`
3. `sqlc generate` — régénère le code dans `backend/common`, `backend/admin`, `backend/student`

---

## Compilation (images Docker)

Les deux applications utilisent un build Docker **multi-stage** avec deux cibles (`--target`) :

| Cible | Usage |
|---|---|
| `prod` | Image de production (binaire strippé, sans symboles de debug) |
| `local` | Image de développement local (symboles de debug + Delve pour le débogage à distance) |

### Stages du build (exemple admin)

| Stage | Image de base | Rôle |
|---|---|---|
| `go-base` | `golang:1.25-alpine` | Sources Go partagées |
| `backend-prod` | `go-base` | Compile le binaire Go optimisé |
| `backend-local` | `go-base` | Compile le binaire Go avec symboles de debug |
| `dlv-builder` | `golang:1.25-alpine` | Compile le débogueur Delve |
| `frontend-builder` | `node:22-alpine` | Build le frontend React (`npm run build`) |
| `base` | `alpine:3.21` | Assemble binaire + fichiers statiques + nginx |
| `prod` / `local` | `base` | Image finale selon la cible |

Le binaire est compilé avec les métadonnées de version injectées via ldflags :

```
go build -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"
```

`VERSION` est dérivé de `git describe --tags --always --dirty`.

### Ports exposés par les conteneurs

| Conteneur | Port applicatif (nginx) | Port debug (Delve) |
|---|---|---|
| `rex-admin` | 8121 | 2345 |
| `rex-eleve` | 8131 | 2346 |

nginx écoute directement sur le port applicatif à l'intérieur du conteneur (pas de remapping).

---

## Application mobile (Expo)

L'application mobile est développée avec **Expo** (React Native). Elle cible trois plateformes : Android, iOS et web. Le point d'entrée est `apps/mobile/`.

### Prérequis

```bash
cd apps/mobile
npm install
```

### Variable d'environnement

L'URL de l'API élève est configurée via la variable `EXPO_PUBLIC_API_BASE` :

```bash
EXPO_PUBLIC_API_BASE=https://vecu-etudiant-eleves-2.mines-ales.fr/api/v2
```

> Dans le build Docker élève, cette variable est fixée à `/api/v2` (chemin relatif, routé par nginx).

### Lancement en développement

```bash
npx expo start           # démarre le serveur Expo (QR code pour Android/iOS)
npx expo start --web     # démarre en mode navigateur
```

### Build Web

Génère les fichiers statiques dans `dist/` :

```bash
npx expo export --platform web
```

Le contenu de `dist/` peut ensuite être servi par n'importe quel serveur HTTP statique (nginx, etc.).

### Build Android

Requiert Android Studio et le SDK Android installés.

```bash
npx expo run:android    # build debug (émulateur ou appareil connecté)
```

Pour un build de production autonome (sans serveur Expo), utiliser **EAS Build** :

```bash
npm install -g eas-cli
eas build --platform android
```

### Build iOS

Requiert macOS et Xcode installés.

```bash
npx expo run:ios        # build debug (simulateur ou appareil connecté)
```

Pour un build de production (distribution App Store) :

```bash
eas build --platform ios
```

> EAS Build délègue la compilation à l'infrastructure cloud d'Expo, ce qui permet de builder une app iOS depuis Linux ou Windows.

---

## Publication des images

Les images sont publiées sur GHCR sous l'organisation `ales-imt` :

- `ghcr.io/ales-imt/rex-admin:latest`
- `ghcr.io/ales-imt/rex-eleve:latest`

```bash
make push
```

Le script `infras/run/push-container.sh` effectue dans l'ordre :
1. Génération `sqlc` (les trois modules : `common`, `admin`, `student`)
2. Build des deux images Docker (target `prod`)
3. Connexion à GHCR avec le token `GITHUB_DOCKER_TOKEN` issu de `secrets-prod.env`
4. Tag et push des images vers GHCR

---

## Déploiement en production

Le déploiement est géré par **Ansible** depuis le poste de développement.

### Prérequis

- Accès SSH aux VMs prod configuré (utilisateur `userdfx`)
- Fichier `.vscode/secrets-prod.env` renseigné
- Certificats TLS déposés dans `infras/ansible/files/` (voir ci-dessous)

### Certificats TLS (`infras/ansible/files/`)

Le rôle Ansible `nginx_vhost` copie le bundle de certificat depuis la machine locale vers chaque VM. Le répertoire `infras/ansible/files/` doit contenir **un sous-répertoire par site**, nommé exactement comme le nom d'hôte (`server_name`) :

```
infras/ansible/files/
├── vecu-etudiant-admin-2.mines-ales.fr/
│   └── Cert_bundle.pem
└── vecu-etudiant-eleves-2.mines-ales.fr/
    └── Cert_bundle.pem
```

Si un répertoire ou un `Cert_bundle.pem` est absent, la tâche *"Déployer le bundle de certificat"* échoue et le déploiement s'arrête.

> Ces fichiers ne sont pas dans git (secrets). Les obtenir auprès de l'équipe ou les générer depuis l'autorité de certification du projet.

### VMs cibles

| Rôle | Hôte | Port applicatif |
|---|---|---|
| Admin | `vecu-etudiant-admin-2.mines-ales.fr` | 8121 |
| Élève | `vecu-etudiant-eleves-2.mines-ales.fr` | 8131 |

### Commandes de déploiement

```bash
make deploy-vm-prod          # déploie admin + élève
make deploy-vm-admin-prod    # déploie uniquement l'admin
make deploy-vm-eleve-prod    # déploie uniquement l'élève
```

Chaque cible génère d'abord le `vault-vars.yml` depuis `secrets-prod.env`, puis lance le playbook Ansible correspondant.

```bash
# Équivalent manuel :
cd infras/ansible
./generate-vault-vars.sh
ansible-playbook -i inventory/prod playbook-admin.yml --extra-vars "@vault-vars.yml" --ask-become-pass
ansible-playbook -i inventory/prod playbook-eleve.yml --extra-vars "@vault-vars.yml" --ask-become-pass
```

> Le mot de passe demandé est le mot de passe `sudo` (`--ask-become-pass`), pas le mot de passe vault.

### Ce que fait Ansible

Chaque playbook applique trois rôles dans l'ordre :

1. **`common`** — configuration système de base de la VM
2. **`podman_app`** — dépose la configuration et le fichier de secrets, puis exécute le script de déploiement (`deploy-admin.sh` / `deploy-eleve.sh`) qui pull l'image depuis GHCR et lance le conteneur avec Podman
3. **`nginx_vhost`** — configure le reverse proxy nginx (TLS, virtualhost)

### Gestion des secrets

Les secrets de production sont dans `.vscode/secrets-prod.env`. Le script `generate-vault-vars.sh` les exporte dans `vault-vars.yml` pour Ansible. Ce fichier contient notamment :

- Credentials PostgreSQL et MariaDB
- Secrets JWT admin et élève
- Token GHCR (`GITHUB_DOCKER_TOKEN`)
- Clé API du serveur IA (`RACK_API_KEY`)
- Clé publique `age` (`AGE_PUBLIC_KEY`)

---

## Coffre numérique (age)

Le projet utilise [`age`](https://age-encryption.org/) pour chiffrer des données sensibles.

```bash
# Installation
sudo apt install age

# Génération d'une paire de clés
age-keygen -o cle_privee.txt
# La clé publique est affichée et incluse dans cle_privee.txt

# Protéger la clé privée par passphrase
age-keygen | age -p > cle_privee_chiffree.age

# Déchiffrer des données
age -d cle_privee_chiffree.age | age -d -i - donnees_a_dechiffrer.age
age --decrypt -i cle_privee.txt fichier.age
```

---

## Workflow complet (prod)

### Premier déploiement (migration incluse)

```bash
make first-deploy-prod
```

Enchaîne dans l'ordre :
1. `migration-prod` — migration v0 → v1 (**à ne lancer qu'une seule fois**)
2. `release-prod` — suite du workflow standard (voir ci-dessous)

> **Attention** : `migration-prod` est à usage unique. Elle doit être supprimée du `makefile.prod` après la première utilisation.

### Déploiements suivants

```bash
make release-prod
```

Enchaîne dans l'ordre :
1. `db-to-code-prod` — extraction du schéma et génération sqlc
2. `build-container-prod` — build des images Docker (target prod)
3. `push` — génération sqlc + push vers GHCR
4. `deploy-vm-prod` — génération vault + déploiement Ansible (admin + élève)

Ou étape par étape :

```bash
make build-container-prod    # Build uniquement les images
make push                    # Build + push vers GHCR
make deploy-vm-prod          # Déploiement Ansible admin + élève
make deploy-vm-admin-prod    # Déploiement Ansible admin uniquement
make deploy-vm-eleve-prod    # Déploiement Ansible élève uniquement
```