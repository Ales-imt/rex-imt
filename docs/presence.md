# Présences — Garanties et valeur juridique

## Objet du document

Ce document explique comment REX-IMT garantit qu'un enregistrement de présence est **authentique, infalsifiable et opposable**, et situe ce mécanisme dans le cadre juridique de la signature électronique.

---

## 1. Ce qu'un enregistrement de présence doit prouver

Pour qu'un relevé de présence soit opposable à un étudiant, trois questions doivent trouver une réponse vérifiable :

| Question | Ce qu'il faut prouver |
|---|---|
| **Qui ?** | L'identité de l'étudiant est certaine au moment du pointage |
| **Quand ?** | L'heure du pointage est fixée par un tiers de confiance, non modifiable après coup |
| **Intégrité** | L'enregistrement n'a pas été altéré depuis sa création |

---

## 2. Le processus de pointage, étape par étape

```
[Étudiant — application mobile]
        │
        │ 1. Authentification LDAP (identifiants institutionnels)
        │    → JWT signé (HS256, secret admin)
        │
        │ 2. Scan du QR code de la séance
        │    → token de séance + JWT envoyés en HTTPS au backend étudiant
        │
[Backend student — Go]
        │
        │ 3. Vérification JWT → extraction user_id (login LDAP → ID base)
        │    Vérification token séance → extraction seance_id, statut (présent/retard)
        │
        │ 4. Ouverture d'une transaction PostgreSQL (pgx.Tx)
        │    ┌─────────────────────────────────────────────────┐
        │    │ 4a. INSERT pointage (seance_id, user_id, statut)│
        │    │     RETURNING pointe_at (timestamp DB)          │
        │    │                                                 │
        │    │ 4b. pg_advisory_xact_lock(9_001_001)            │
        │    │     → sérialisation des écritures ledger        │
        │    │                                                 │
        │    │ 4c. SELECT hash FROM presence_ledger            │
        │    │     ORDER BY seq DESC LIMIT 1                   │
        │    │     → prev_hash                                 │
        │    │                                                 │
        │    │ 4d. recorded_at = time.Now().UTC() (nanoseconde)│
        │    │     hash = SHA-256(seance_id|user_id|statut|    │
        │    │             event_at|recorded_at|prev_hash)     │
        │    │                                                 │
        │    │ 4e. INSERT presence_ledger                      │
        │    │     RETURNING seq                               │
        │    └─────────────────────────────────────────────────┘
        │    COMMIT (4a + 4b + 4c + 4d + 4e atomiques)
        │
        ▼
[presence_ledger — PostgreSQL]
        │
        │ 5. (asynchrone — cron 02h00) POST /presence/ledger/anchor
        │    → hash du dernier maillon envoyé à FreeTSA (RFC 3161)
        │    → jeton horodaté stocké dans presence_anchor
```

---

## 3. Les trois couches de garantie

### 3.1 Identification (Qui ?)

L'étudiant s'authentifie avec ses **identifiants institutionnels LDAP** (annuaire de l'école). Le serveur émet un JWT signé liant `user_id` à la session. Aucun tiers ne peut usurper l'identité d'un étudiant sans connaître son mot de passe LDAP.

Limite : REX-IMT ne vérifie pas la présence physique (pas de biométrie). Le système prouve que **l'identifiant numérique de l'étudiant a été utilisé** pour scanner le QR code depuis l'application, à l'heure considérée.

### 3.2 Intégrité de la chaîne (Infalsifiabilité)

Chaque enregistrement contient le hash du précédent :

```
hash(N) = SHA-256( seance_id | user_id | statut | event_at | recorded_at | prev_hash(N-1) )
```

Modifier un enregistrement (serait-ce par un accès direct à la base) invalide son hash et **casse la chaîne à partir de ce point**. La vérification via `GET /api/v2/presence/ledger/verify` détecte immédiatement le premier maillon altéré.

La contrainte `UNIQUE` sur la colonne `hash` et les FK `ON DELETE RESTRICT` sur `seance_id` et `user_id` empêchent toute suppression silencieuse.

### 3.3 Horodatage externe (Quand ?)

L'ancrage RFC 3161 confie à une autorité d'horodatage tierce (FreeTSA) la preuve que le hash `H` existait **avant** l'heure indiquée dans le jeton. Le jeton CMS signé est archivé dans `presence_anchor` avec le certificat racine de la TSA, garantissant la vérification hors ligne.

---

## 4. Cadre juridique

### 4.1 Textes applicables

| Texte | Portée |
|---|---|
| **Règlement eIDAS** (UE 910/2014) | Définit les niveaux de signature électronique et leur valeur dans l'UE |
| **Code civil, art. 1366** | L'écrit électronique a la même force probante que le papier si l'identité est assurée et l'intégrité garantie |
| **Code civil, art. 1367** | La signature électronique consiste en un procédé fiable d'identification garantissant son lien avec l'acte |
| **Code de l'éducation, art. L123-1** | L'assiduité est une obligation légale pour les étudiants inscrits en formation initiale |

### 4.2 Niveau de signature au sens eIDAS

eIDAS distingue trois niveaux :

| Niveau | Exigences | Valeur |
|---|---|---|
| **Simple (SES)** | Toute donnée électronique logiquement associée au signataire | Recevable, appréciée librement par le juge |
| **Avancée (SEA)** | Liée univoquement au signataire, détection de toute modification, données sous contrôle exclusif | Force probante renforcée |
| **Qualifiée (SEQ)** | SEA + dispositif certifié + certificat qualifié délivré par un prestataire de confiance de la liste EU | Présomption légale de fiabilité (art. 25 eIDAS) |

**REX-IMT fournit une signature électronique de niveau avancé (SEA)** :

- *Liée univoquement* — le JWT est émis après authentification LDAP ; `user_id` est non ambiguë.
- *Détection de modification* — le hash chaining détecte toute altération postérieure.
- *Sous contrôle exclusif* — seul l'étudiant connaît son mot de passe LDAP ; le JWT a une durée de vie courte (15 min).

Le système n'atteint pas le niveau **qualifié** : il n'existe pas de certificat qualifié individuel par étudiant émis par un prestataire de confiance qualifié (QTSP). Une SEQ exigerait l'intégration d'un service de signature qualifiée (ex. DocuSign, Universign, FranceConnect+).

### 4.3 Horodatage

L'horodatage RFC 3161 fourni par FreeTSA est **cryptographiquement valide** (SHA-256, jeton CMS signé, vérifiable hors ligne). FreeTSA n'est pas inscrite sur la liste de confiance UE (EU Trusted List) — ses jetons ne constituent donc pas un **horodatage qualifié** au sens de l'art. 42 eIDAS.

Pour un usage devant juridiction, les jetons FreeTSA renforcent le faisceau de preuves sans constituer à eux seuls une preuve irréfragable. Pour une valeur absolue, remplacer ou compléter par une TSA qualifiée (liste disponible sur [https://esignature.ec.europa.eu/efda/tl-browser/](https://esignature.ec.europa.eu/efda/tl-browser/)).

### 4.4 Valeur probatoire et charge de la preuve

Le dispositif produit un **faisceau de preuves concordantes** :

1. Log d'authentification LDAP (identité)
2. Enregistrement `pointage` (horodatage DB)
3. Maillon `presence_ledger` (hash, chaîne ininterrompue)
4. Jeton RFC 3161 (antériorité certifiée par un tiers)

En cas de contestation par un étudiant, le dispositif renverse la charge de la preuve en pratique : l'étudiant devrait démontrer que son identifiant LDAP a été utilisé à son insu **et** que la chaîne de hashes n'a pas été altérée. Cette démonstration est cryptographiquement impossible sans compromettre l'annuaire LDAP de l'école.

### 4.5 Base légale RGPD

| Donnée | Base légale | Durée |
|---|---|---|
| `pointage` (user_id, seance_id, statut, heure) | Obligation légale — assiduité (art. L123-1 Code Éducation) | Durée scolarité + 5 ans |
| `presence_ledger` (hash, prev_hash) | Même obligation — intégrité du registre | Idem |
| `presence_anchor` (jeton RFC 3161) | Même obligation — opposabilité | Idem |

Seul le hash SHA-256 transite vers la TSA externe. Aucune donnée nominative ne quitte l'établissement lors de l'ancrage.

La suppression d'un compte étudiant est bloquée par les FK `RESTRICT` sur `presence_ledger`. La procédure RGPD consiste à **anonymiser** les données nominatives dans `users` et `pointage` (pseudonymisation) sans supprimer les maillons du ledger, préservant ainsi l'intégrité de la chaîne.

---

## 5. Limites et vecteurs de risque résiduels

| Risque | Niveau | Mitigation |
|---|---|---|
| Partage de mot de passe LDAP entre étudiants | Organisationnel | Charte informatique ; impossibilité de prouver le partage a posteriori |
| Accès physique à l'application mobile d'un autre étudiant | Organisationnel | JWT de 15 min ; déverrouillage du téléphone requis |
| Compromission de la base PostgreSQL | Technique | La chaîne cassée est détectable ; FK RESTRICT empêche la suppression silencieuse |
| Compromission du serveur Go | Technique | `recorded_at` fixé côté serveur, advisory lock ; altération visible dans les logs |
| FreeTSA hors ligne lors de l'ancrage | Disponibilité | Plusieurs URLs TSA configurables ; l'absence d'ancrage ne casse pas la chaîne |

---

## 6. Opérations d'administration

### Vérifier l'intégrité du registre

```bash
curl -H "Authorization: Bearer <token_admin>" \
     https://vecu-etudiant-admin-2.mines-ales.fr/api/v2/presence/ledger/verify
```

Réponse : `{"ok": true}` ou `{"ok": false, "broken_at": <seq>, "error": "..."}`.

### Ancrer manuellement

```bash
curl -X POST -H "Authorization: Bearer <token_admin>" \
     https://vecu-etudiant-admin-2.mines-ales.fr/api/v2/presence/ledger/anchor
```

### Télécharger le certificat FreeTSA

```bash
make fetch-freetsa-cert
# Vérifier l'empreinte affichée sur https://freetsa.org avant de faire confiance.
```

---

*Document — Juin 2026 — REX-IMT / Mines Alès IMT*
