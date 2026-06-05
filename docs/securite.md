# Audit de sécurité — REX-IMT

**Date :** Juin 2026
**Périmètre :** Code source complet (`backend/`, `apps/`, `infras/`)
**Score global : 6.2 / 10**

---

## Synthèse

| Sévérité | Nombre |
|---|---|
| 🔴 Critique | 2 |
| 🟠 Élevée | 5 |
| 🟡 Moyenne | 4 |
| ✅ Points positifs | 6 |

---

## 🔴 Vulnérabilités critiques

### CRIT-1 — Backdoor de développement en production

**Fichier :** `backend/common/pkg/auth/ldap.go` — ligne 17

**Description :**
Un compte de test codé en dur permet de contourner totalement l'authentification LDAP avec le mot de passe `"t"`. Ce code est compilé dans les deux applications (admin et élève) et est actif en production.

```go
// ldap.go — ligne 17
if identifiant == "clement.trens@etu.mines-ales.fr" && password == "t" {
    return &identity, nil  // bypass total de l'authentification
}
```

**Correction :** Supprimer ces 4 lignes immédiatement. Ce type de bypass ne doit jamais exister hors d'un environnement de test isolé, et certainement pas dans le module partagé `common`.

---

### CRIT-2 — Clé privée age committée dans le dépôt Git

**Fichier :** `.vscode/cle_privee_age-dev.txt`

**Description :**
La clé privée age de développement est présente dans le dépôt Git. Elle permet de déchiffrer les données techniques (IP + student_id) stockées dans la strongbox de tous les feedbacks de développement. Quiconque a accès au dépôt (présent ou passé via l'historique) peut déchiffrer ces données.

```
# .vscode/cle_privee_age-dev.txt — présent dans le dépôt
AGE-SECRET-KEY-1DTDCLWJU2V257XVKFLA95H0W...
```

Le `.gitignore` couvre `cle_privee.txt` mais **pas** `cle_privee_age-dev.txt`.

**Correction :**
1. Régénérer immédiatement une nouvelle paire de clés age.
2. Ajouter `cle_privee*.txt` au `.gitignore` (pattern plus large).
3. Purger l'historique Git avec `git filter-repo --path .vscode/cle_privee_age-dev.txt --invert-paths`.
4. Considérer les données de dev potentiellement compromises.

---

## 🟠 Vulnérabilités élevées

### HIGH-1 — Authentification LDAP sans TLS

**Fichier :** `backend/common/pkg/auth/ldap.go`, templates Ansible (`group_vars/admin.yml`, `group_vars/eleve.yml`)

**Description :**
La connexion au serveur LDAP de l'école utilise `ldap://` (port 389) sans chiffrement. Les identifiants des utilisateurs transitent en clair entre l'application et l'annuaire.

```yaml
# Config production (template Ansible)
ldap:
  url: ldap://ldap.mines-ales.fr:389  # pas de TLS
```

**Correction :** Passer à `ldaps://ldap.mines-ales.fr:636` ou activer STARTTLS (`ldap.StartTLS()`). Vérifier avec la DSI que le serveur LDAP de l'école supporte ces protocoles.

---

### HIGH-2 — JWT secret identique admin et élève en développement

**Fichier :** `backend/admin/cmd/config.yaml`, `backend/student/cmd/config.yaml`

**Description :**
Les configurations de développement utilisent la même variable `${JWT_SECRET}` pour les deux applications. Un token élève signé en dev pourrait être accepté par le backend admin si les secrets sont identiques. En production, ce point est corrigé (`jwt_secret_admin` / `jwt_secret_eleve`), mais l'environnement dev reste risqué.

```yaml
# config.yaml dev — même variable pour admin ET élève
jwt:
  secret: ${JWT_SECRET}  # identique des deux côtés
```

**Correction :** Utiliser `${JWT_SECRET_ADMIN}` et `${JWT_SECRET_ELEVE}` également dans les configs dev, comme c'est déjà le cas en production via Ansible.

---

### HIGH-3 — Middleware FullLogRequest logue le body complet (mots de passe)

**Fichier :** `backend/common/pkg/services/full_logger_request.go`

**Description :**
Le middleware `FullLogRequest` logue intégralement le body de toutes les requêtes HTTP, y compris les identifiants et mots de passe LDAP envoyés sur `POST /auth/login`. Il est actuellement commenté dans `main.go`, mais présente un risque élevé de réactivation accidentelle lors d'un débogage.

```go
// full_logger_request.go
bodyBytes, _ := io.ReadAll(r.Body)
fmt.Printf("Body: %s\n", string(bodyBytes))
// → logue {"identifiant":"...", "password":"..."}
```

**Correction :** Supprimer ce middleware ou le remplacer par une version qui filtre explicitement les champs sensibles (`password`, `secret`, etc.). Ne jamais l'activer sur une route d'authentification.

---

### HIGH-4 — Refresh token avec durée de vie excessive (90 jours)

**Fichier :** `backend/admin/cmd/config.yaml`, `backend/student/cmd/config.yaml`

**Description :**
Le refresh token expire au bout de 2160 heures (90 jours). Un token volé (via XSS, fuite de logs, compromission d'appareil) reste exploitable pendant 3 mois.

```yaml
jwt:
  accessTokenExpiresIn: 15m      # ✅ correct
  refreshTokenExpiresIn: 2160h   # 90 jours — trop long
```

**Correction :** Réduire à 7 ou 14 jours maximum. Mettre à jour en conséquence la mention "3 mois" dans la page "À propos" de l'application mobile (`apropos-content.tsx`).

---

### HIGH-5 — Appels HTTP en clair vers webdfd.mines-ales.fr

**Fichier :** `backend/student/cmd/main.go` — lignes 74–82

**Description :**
L'application élève récupère les données de planning, promotions et cours via HTTP non chiffré. Ces données (noms d'étudiants, emplois du temps, groupes) transitent en clair sur le réseau.

```go
ElevesURL:   "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=eleves_txt"
PlanningURL: "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe"
```

**Correction :** Passer en `https://` si le serveur le supporte. Sinon, s'assurer que ces communications restent strictement sur le réseau interne de l'école (non joignable depuis Internet).

---

## 🟡 Vulnérabilités moyennes

### MED-1 — Aucune validation de taille sur le contenu des feedbacks

**Fichier :** `backend/student/pkg/feedback/feedback.go`

**Description :**
Le champ `Content` des feedbacks n'a pas de limite de taille côté backend. Un utilisateur malveillant pourrait soumettre des feedbacks de plusieurs Mo, saturant la base de données ou la mémoire du processeur lors de la classification IA.

**Correction :**
```go
const maxFeedbackLength = 5000

func (f *FeedbackRequest) validate() error {
    if len(f.Content) > maxFeedbackLength {
        return fmt.Errorf("contenu trop long (max %d caractères)", maxFeedbackLength)
    }
    if len(f.Pseudo) > 100 { return fmt.Errorf("pseudo trop long") }
    if len(f.Promotion) > 100 { return fmt.Errorf("promotion trop longue") }
    return nil
}
```

---

### MED-2 — Ollama exposé sans authentification sur le réseau local (dev)

**Fichier :** `infras/container/compose.yaml`

**Description :**
En environnement de développement, le conteneur Ollama expose le port 11434 sur `0.0.0.0` sans authentification. Tout utilisateur du réseau local (ou d'un réseau partagé) peut envoyer des requêtes arbitraires au LLM.

```yaml
ollama:
  ports:
    - "11434:11434"  # exposé sur toutes les interfaces
```

**Correction :** Lier le port à l'interface locale uniquement :

```yaml
ports:
  - "127.0.0.1:11434:11434"
```

---

### MED-3 — Absence de headers de sécurité HTTP sur le vhost nginx

**Fichier :** `infras/ansible/roles/nginx_vhost/templates/vhost.conf.j2`

**Description :**
Le template nginx de production ne définit aucun header de sécurité. Les applications sont exposées au clickjacking, au MIME sniffing, et ne disposent pas de politique CSP ni de HSTS.

**Correction — ajouter dans le bloc `server` HTTPS :**

```nginx
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline';" always;
```

---

### MED-4 — Rate limiting uniquement sur /auth/login

**Fichier :** `backend/common/pkg/auth/ratelimit.go`, `backend/common/pkg/auth/routes.go`

**Description :**
Le rate limiter (5 requêtes/minute/IP) ne couvre que le endpoint `POST /auth/login`. Les endpoints `/auth/refresh` et `/auth/me` ne sont pas protégés contre les abus en volume.

**Correction :** Étendre le middleware ou créer un rate limiter plus permissif appliqué à toutes les routes `/auth/*` :

```go
// routes.go
r.Route("/auth", func(r chi.Router) {
    r.With(RateLimitLogin).Post("/login", ...)
    r.With(RateLimitGeneral).Post("/refresh", ...)  // à ajouter
})
```

---

## ✅ Points positifs

| Point | Détail |
|---|---|
| Injection LDAP impossible | `ldap.EscapeFilter()` utilisé systématiquement sur tous les filtres LDAP |
| Refresh tokens hashés en base | SHA-256 appliqué avant stockage — jamais stockés en clair |
| Vérification de session active | Le middleware `Security` vérifie l'existence du refresh token en base à chaque requête |
| Secrets production via Ansible vault | `vault-vars.yml` chiffré et dans le `.gitignore` — non versionné |
| Chiffrement age pour la strongbox | Chiffrement asymétrique X25519 pour les données d'identification (IP + student_id) |
| Purge RGPD automatique | Anonymisation après 1 an, suppression après 3 ans, exécution toutes les 24h |

---

## Plan d'action recommandé

### Immédiat (< 24h)

1. **Supprimer le backdoor** `clement.trens` dans `ldap.go`
2. **Révoquer et régénérer** la clé privée age de dev, purger l'historique Git

### Court terme (< 1 semaine)

3. Activer **LDAPS** pour la connexion à l'annuaire
4. Aligner les secrets JWT dev sur la configuration prod (`JWT_SECRET_ADMIN` / `JWT_SECRET_ELEVE`)
5. **Supprimer** le middleware `FullLogRequest` ou le rendre inoffensif
6. Réduire la durée des refresh tokens à **14 jours**

### Moyen terme (< 1 mois)

7. Passer les appels `webdfd.mines-ales.fr` en **HTTPS**
8. Ajouter une **validation de taille** sur les inputs feedback
9. Ajouter les **headers de sécurité** nginx (CSP, X-Frame-Options, HSTS)
10. Étendre le **rate limiting** à toutes les routes `/auth/*`
11. Restreindre le port Ollama à `127.0.0.1` en dev

---

*Rapport généré sur la base de l'analyse statique du code source du dépôt `github.com/Ales-imt/rex-imt` — Juin 2026*