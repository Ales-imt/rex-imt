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

---

## Registre d'intégrité des présences

### Architecture

La table `pointage` enregistre les présences ; elle ne possède pas d'identifiant ordonnable et peut être modifiée. Pour garantir l'inaltérabilité des enregistrements, une couche **append-only** est ajoutée par-dessus :

| Table | Rôle |
|---|---|
| `presence_ledger` | Un maillon par pointage certifié ; séquence croissante stricte |
| `presence_anchor` | Jetons RFC 3161 scellant l'extrémité de la chaîne auprès d'une TSA |

### Chaînage cryptographique

**Formule :**
```
hash(N) = SHA-256( seance_id | user_id | statut | event_at | recorded_at | prev_hash )
```

- Les champs sont concaténés avec `|` dans un **ordre fixe et contractuel** (voir [hash.go](../backend/common/pkg/ledger/hash.go)).
- Les horodatages sont en RFC3339Nano UTC (précision nanoseconde).
- `prev_hash` du premier maillon = `0000...0` (64 zéros, sentinel Genesis).
- `recorded_at` est toujours positionné côté serveur Go (`time.Now().UTC()`) — jamais fourni par le client.
- L'insertion utilise `pg_advisory_xact_lock` pour sérialiser les écritures concurrentes : aucun deux maillons ne peuvent partager le même prédécesseur.

**Atomicité :** le pointage et son maillon sont insérés dans la **même transaction pgx** (backend student, `PostPointage`).

**Vérification :** `GET /presence/ledger/verify` (admin) relit toute la chaîne, recalcule chaque hash et signale le premier `seq` cassé.

### Ancrage externe RFC 3161

**But :** protéger l'extrémité de la chaîne (le dernier maillon n'a pas encore de successeur). Un ancrage périodique le scelle auprès d'une autorité d'horodatage (TSA).

**RGPD :** seul le hash SHA-256 transite vers la TSA. Aucune donnée nominative ne quitte le système. Base légale : obligation d'assiduité (art. L123-1 Code Éducation).

**Déclenchement :**
- Automatique : configurer un cron externe ou une tâche planifiée appelant `POST /presence/ledger/anchor` selon `anchorCron` (ex. `"0 2 * * *"`).
- Manuel : `POST /api/v2/presence/ledger/anchor` (rôle admin ou gestionnaire).

**TSA par défaut :** `https://freetsa.org/tsr`. Plusieurs TSA peuvent être configurées (`presence.timestamp.urls`) pour la redondance — un jeton est archivé par TSA.

**Certificat racine FreeTSA :** télécharger depuis `https://freetsa.org/files/cacert.pem` et placer dans `./x509/freetsa/cacert.pem` (chemin configuré par `caCertPath`). Ce certificat est archivé avec chaque jeton dans `presence_anchor.tsa_cert`, garantissant la vérifiabilité hors ligne si la TSA disparaît.

**Durée de conservation :** les maillons du registre et les jetons d'ancrage sont conservés aussi longtemps que les obligations légales de tenue du registre d'assiduité l'exigent (au moins la durée de la scolarité + 5 ans). La suppression d'un utilisateur est bloquée par la FK RESTRICT tant que des maillons existent — la procédure RGPD consiste à anonymiser les données nominatives dans `user` et `pointage` sans supprimer les hash du ledger.

**Vérification des ancres :** `GET /api/v2/presence/ledger/verify` renvoie l'état de chaque ancre (signature CMS vérifiée, hash correspondant au `anchored_hash`).

### Placer le certificat FreeTSA

```bash
mkdir -p backend/admin/x509/freetsa
curl -o backend/admin/x509/freetsa/cacert.pem https://freetsa.org/files/cacert.pem
# Vérifier l'empreinte publiée sur https://freetsa.org avant de faire confiance.
```

---

## Plan d'action

| Priorité | Finding | Effort | Dépendance |
|---|---|---|---|
| 🟠 Court terme | LDAP sans TLS (HIGH-1) | Faible — 1 ligne de config | DSI Mines Alès |
| 🟠 Court terme | Refresh token 90j (HIGH-2) | Faible — 1 ligne de config + apropos | Aucune |
| 🟡 Moyen terme | HTTP webdfd (MED-1) | Faible — 4 URLs | DSI Mines Alès |

---

*Rapport généré sur la base de l'analyse statique du code source — commit `73a7ca8` — `github.com/Ales-imt/rex-imt` — Juin 2026*