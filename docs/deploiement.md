# REX-IMT — Compilation et déploiement

## Présentation

REX-IMT est une plateforme d'évaluation composée de trois applications :

- **Admin** : interface de gestion à destination des enseignants (backend Go + frontend React)
- **Élève web** : interface web à destination des étudiants (backend Go)
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
│   ├── container/          # Docker Compose (environnement dev)
│   ├── run/                # Dockerfiles, scripts de lancement, configs nginx
│   ├── liquibase/          # Migrations de base de données
│   ├── migration/          # Outil Go de migration des données
│   └── ansible/            # Déploiement sur les VMs prod
├── makefile                # Point d'entrée principal
├── makefile.dev            # Cibles dev
└── makefile.prod           # Cibles prod
```

---

## Compilation

Les deux applications utilisent un build Docker **multi-stage** défini dans `infras/run/Dockerfile.admin` et `infras/run/Dockerfile.eleve`.

### Étapes du build (exemple admin)

| Stage | Image de base | Rôle |
|---|---|---|
| `backend-builder` | `golang:1.25-alpine` | Compile le binaire Go |
| `frontend-builder` | `node:22-alpine` | Build le frontend React (`npm run build`) |
| Image finale | `alpine:3.21` | Assemble binaire + fichiers statiques + nginx |

Le binaire est compilé avec les métadonnées de version :

```
go build -ldflags="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"
```

`VERSION` est dérivé de `git describe --tags`.

### Lancement local (dev)

```bash
make start          # lance admin + élève + LDAP en conteneurs Docker locaux
make start-admin    # lance uniquement l'admin
make start-eleve    # lance uniquement l'élève
make stop           # arrête tout
```

Les variables de connexion sont lues depuis `.vscode/secrets-dev.env`.

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
npx expo run:android                   # build debug (émulateur ou appareil connecté)
npx expo build:android --type apk      # build release APK (nécessite un compte Expo)
```

Pour un build de production autonome (sans serveur Expo), utiliser **EAS Build** :

```bash
npm install -g eas-cli
eas build --platform android
```

### Build iOS

Requiert macOS et Xcode installés.

```bash
npx expo run:ios                       # build debug (simulateur ou appareil connecté)
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

Ce script (`infras/run/push.sh`) :
1. Build les deux images Docker
2. Se connecte à GHCR avec le token `GITHUB_DOCKER_TOKEN` issu de `secrets-prod.env`
3. Tag et push les images

---

## Déploiement en production

Le déploiement est géré par **Ansible** depuis le poste de développement.

### Prérequis

- Accès SSH aux VMs prod configuré
- Fichier `infras/ansible/vault-vars.yml` déchiffrable (mot de passe vault)

### VMs cibles

| Rôle | Hôte | Port applicatif |
|---|---|---|
| Admin | `vecu-etudiant-admin-2.mines-ales.fr` | 8121 |
| Élève | `vecu-etudiant-eleves-2.mines-ales.fr` | 8131 |

### Commande de déploiement

```bash
make deploy-prod
# équivalent à :
# ansible-playbook -i inventory/prod site.yml --extra-vars "@vault-vars.yml" --ask-vault-pass
```

Pour déployer uniquement un serveur :

```bash
cd infras/ansible
ansible-playbook -i inventory/prod playbook-admin.yml --extra-vars "@vault-vars.yml" --ask-vault-pass
ansible-playbook -i inventory/prod playbook-eleve.yml --extra-vars "@vault-vars.yml" --ask-vault-pass
```

### Ce que fait Ansible

Chaque playbook applique trois rôles dans l'ordre :

1. **`common`** — configuration système de base de la VM
2. **`podman_app`** — dépose la configuration et le fichier de secrets, puis exécute le script de déploiement (`deploy-admin.sh` / `deploy-eleve.sh`) qui pull l'image depuis GHCR et lance le conteneur avec Podman
3. **`nginx_vhost`** — configure le reverse proxy nginx (TLS, virtualhost)

### Gestion des secrets

Les secrets (mots de passe BDD, JWT, tokens) sont stockés dans `vault-vars.yml`, chiffré avec `ansible-vault`. Pour régénérer ce fichier à partir de `.vscode/secrets-prod.env` :

```bash
cd infras/ansible
./generate-vault-vars.sh
ansible-vault encrypt vault-vars.yml
```

---

## Workflow complet (prod)

```
make push           # 1. Build + push des images sur GHCR
make deploy-prod    # 2. Ansible déploie les images sur les VMs
```

Ou en une seule commande :

```bash
make infra-prod
```
