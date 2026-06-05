# Audit de sécurité — REX-IMT

**Date :** Juin 2026 | **Commit :** `73a7ca8` | **Périmètre :** Code source complet (`backend/`, `apps/`, `infras/`)

---

## Synthèse

| Sévérité | Nombre |
|---|---|
| 🔴 Critique | 0 |
| 🟠 Élevée | 2 |
| 🟡 Moyenne | 1 |
| ✅ Points positifs | 12 |

**Score global : 8.4 / 10**

---

## 🟠 Vulnérabilités élevées

### HIGH-1 — Authentification LDAP sans TLS

**Fichier :** `infras/ansible/roles/podman_app/templates/config-admin.yaml.j2`, `config-eleve.yaml.j2`

**Description :**
La connexion au serveur LDAP de l'école utilise `ldap://` (port 389) sans chiffrement. Les identifiants des utilisateurs transitent en clair entre l'application et l'annuaire LDAP à chaque authentification.

```yaml
ldap:
  url: ldap://ldap.mines-ales.fr:389  # pas de TLS
```

**Correction :**
Passer à `ldaps://ldap.mines-ales.fr:636` ou activer STARTTLS. À coordonner avec la DSI de Mines Alès.

```yaml
ldap:
  url: ldaps://ldap.mines-ales.fr:636
```

---

### HIGH-2 — Refresh token avec durée de vie excessive (90 jours)

**Fichier :** `backend/admin/cmd/config.yaml`, `backend/student/cmd/config.yaml`

**Description :**
Le refresh token expire au bout de 2160 heures (90 jours). Un token volé reste exploitable pendant 3 mois sans possibilité de détection automatique.

```yaml
jwt:
  accessTokenExpiresIn: 15m      # ✅ correct
  refreshTokenExpiresIn: 2160h   # 90 jours — trop long
```

**Correction :**
Réduire à 14 jours. Mettre à jour la mention correspondante dans `apps/mobile/components/apropos-content.tsx`.

```yaml
refreshTokenExpiresIn: 336h  # 14 jours
```

---

## 🟡 Vulnérabilités moyennes

### MED-1 — Appels HTTP en clair vers webdfd.mines-ales.fr

**Fichier :** `backend/student/cmd/main.go` — lignes 74–82

**Description :**
Les 4 appels vers le serveur de planning, notes et promotions utilisent HTTP non chiffré. Les données d'étudiants (noms, emplois du temps, groupes, notes) transitent en clair sur le réseau.

```go
ElevesURL:   "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=eleves_txt"
PlanningURL: "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe"
"http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=promos_txt"
"http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=cours_txt"
```

**Correction :**
Passer en `https://` si le serveur le supporte. Sinon, s'assurer que les VMs de production communiquent avec `webdfd` uniquement via le réseau interne de l'école.

---

## ✅ Points positifs

| # | Point | Détail |
|---|---|---|
| 1 | Injection LDAP impossible | `ldap.EscapeFilter()` utilisé systématiquement sur tous les filtres |
| 2 | Refresh tokens hashés en base | SHA-256 — jamais stockés en clair |
| 3 | Vérification de session active | Middleware `Security` — session vérifiée en base à chaque requête |
| 4 | Secrets production via Ansible vault | `vault-vars.yml` chiffré et dans le `.gitignore` |
| 5 | Chiffrement asymétrique age | Strongbox : IP + student_id chiffrés avec clé publique X25519 |
| 6 | Purge RGPD automatique | Anonymisation à 1 an, suppression à 3 ans, exécution toutes les 24h |
| 7 | Headers de sécurité nginx | X-Frame-Options, CSP, HSTS, X-Content-Type-Options, Referrer-Policy |
| 8 | JWT secrets distincts admin / élève | `jwt_secret_admin` / `jwt_secret_eleve` — séparés en dev et prod |
| 9 | Validation taille feedbacks | `maxContentLen = 2000`, Unicode-aware (`utf8.RuneCountInString`) |
| 10 | Pas de logger sensible | Aucun middleware ne logue les corps de requêtes HTTP |
| 11 | Authentification déléguée au LDAP | Aucun mot de passe stocké en base |
| 12 | TLS en transit | Reverse proxy nginx avec TLS 1.2/1.3 — certificats gérés via Ansible |

---

## Plan d'action

| Priorité | Finding | Effort | Dépendance |
|---|---|---|---|
| 🟠 Court terme | LDAP sans TLS (HIGH-1) | Faible — 1 ligne de config | DSI Mines Alès |
| 🟠 Court terme | Refresh token 90j (HIGH-2) | Faible — 1 ligne de config + apropos | Aucune |
| 🟡 Moyen terme | HTTP webdfd (MED-1) | Faible — 4 URLs | DSI Mines Alès |

---

*Rapport généré sur la base de l'analyse statique du code source — commit `73a7ca8` — `github.com/Ales-imt/rex-imt` — Juin 2026*