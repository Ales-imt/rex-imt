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

L'ancrage RFC 3161 confie à une autorité d'horodatage tierce (FreeTSA) la preuve que le hash `H` existait **avant** l'heure indiquée dans le jeton. Le jeton CMS signé est archivé dans `presence_anchor` avec le certificat racine de la TSA, garantissant la vérification hors ligne. L'ancrage automatique tourne toutes les heures et n'ancre que si la tête de chaîne a bougé.

### 3.4 Témoin externe (Résistance à un initié)

Après chaque nouvel ancrage, la tête de chaîne (seq + hash + jeton + certificat TSA) est envoyée par e-mail vers une boîte contrôlée par un tiers, hors de portée d'un administrateur de l'infrastructure. Une chaîne réécrite puis ré-ancrée ne peut pas correspondre aux témoins déjà reçus : l'altération devient **détectable** (pas impossible). Exigences de déploiement et limites : voir [temoin-externe.md](temoin-externe.md).

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
| `pointage` (user_id, seance_id, statut, heure) | Obligation légale — assiduité (art. L123-1 Code Éducation) **et** conservation des pièces justificatives de l'action de formation | Conservées ; identité rattachée anonymisée à sortie + 10 ans |
| `presence_ledger` (hash, prev_hash) | Même obligation — intégrité du registre | Idem — les maillons ne sont jamais supprimés |
| `presence_anchor` (jeton RFC 3161) | Même obligation — opposabilité | Idem |
| `justification` (user_id, plage horaire, saisissant, horodatages, corrections) | Même obligation — l'excuse motive une absence au regard de l'assiduité. **Aucun motif n'est collecté**, donc aucune donnée relevant de l'art. 9 RGPD | Scolarité + 5 ans ; append-only, jamais purgée (voir §5 bis) |

L'horizon de **10 ans** est un plafond conservateur couvrant l'obligation la plus longue applicable aux pièces justificatives d'une action de formation : 3 ans pour le contrôle Qualiopi/DREETS, 6 ans pour le contrôle OPCO et l'obligation fiscale (art. L102-B du LPF, à compter du dernier versement), 10 ans en cas de cofinancement européen FSE+. L'école ne connaissant pas action par action ses financements, le plafond est appliqué uniformément. Le décompte part de la date de sortie lue dans Auréga — simplification documentée et à revalider dans [`rgpd-dpo.md`](rgpd-dpo.md) §7 bis.

Seul le hash SHA-256 transite vers la TSA externe. Aucune donnée nominative ne quitte l'établissement lors de l'ancrage.

#### Mécanisme de fin de vie — anonymisation en place

Un compte porteur de maillons de registre **n'est jamais supprimé** : la FK `fk_presence_ledger_user` est `ON DELETE RESTRICT`, et un `DELETE` qui aboutirait détruirait par ailleurs les pointages, la FK `pointage_user_id_fkey` étant `ON DELETE CASCADE`.

À l'échéance, c'est l'**identité** qui est effacée, **en place**, dans la seule table `user` :

- `name` et `surname` sont vidés, l'email remplacé par un placeholder unique par identifiant, `auth_source` passé à `anonymized` ;
- **l'identifiant entier est conservé** — c'est lui que scelle la fonction de hachage ([`ledger/hash.go`](../backend/common/pkg/ledger/hash.go)) ;
- **ni `presence_ledger`, ni `pointage.user_id`, ni la fonction de hachage ne sont modifiés.** Aucun maillon n'est retiré, aucun pointage n'est perdu, tous les hachés restent recalculables et `VerifyChain` continue de passer.

Le terme employé est bien **anonymisation** et non pseudonymisation : aucune table de correspondance ne permet de retrouver l'identité à partir de l'identifiant conservé. Celui-ci ne survit que comme clé technique de chaînage, sans donnée nominative associée nulle part dans le système.

Implémentation : [`backend/admin/pkg/account/`](../backend/admin/pkg/account/), appelée à la fois par la purge automatique et par le CRUD admin. Après chaque cycle ayant anonymisé au moins un compte, `VerifyChain` est rejoué en garde-fou et toute anomalie journalisée en ALERTE.

---

## 5. Limites et vecteurs de risque résiduels

| Risque | Niveau | Mitigation |
|---|---|---|
| Partage de mot de passe LDAP entre étudiants | Organisationnel | Charte informatique ; impossibilité de prouver le partage a posteriori |
| Accès physique à l'application mobile d'un autre étudiant | Organisationnel | JWT de 15 min ; déverrouillage du téléphone requis |
| Compromission de la base PostgreSQL | Technique | La chaîne cassée est détectable ; FK RESTRICT empêche la suppression silencieuse |
| Initié avec accès base + infra (réécriture puis ré-ancrage) | Technique/Organisationnel | Témoin externe : la chaîne falsifiée ne correspond plus aux témoins archivés chez un tiers ([temoin-externe.md](temoin-externe.md)) |
| Compromission du serveur Go | Technique | `recorded_at` fixé côté serveur, advisory lock ; altération visible dans les logs |
| FreeTSA hors ligne lors de l'ancrage | Disponibilité | Plusieurs URLs TSA configurables ; l'absence d'ancrage ne casse pas la chaîne |
| Altération de l'historique des **excuses** par accès base direct | Technique | Tables append-only par trigger, mais **non chaînées cryptographiquement** : l'altération ne serait pas détectable (voir ci-dessous) |

### 5 bis. Excuses d'absence — un niveau de garantie volontairement moindre

Une **excuse** (`justification`) est un acte administratif d'un gestionnaire d'année portant sur une plage horaire. Elle ne se substitue jamais à un pointage : elle forme une **couche** posée sur l'absence, et le pointage l'emporte toujours — un étudiant excusé qui a scanné reste `PRESENT`. Aucune ligne n'est écrite dans `pointage`, aucun maillon n'est ajouté à `presence_ledger`, et le statut `EXCUSE` n'existe pas dans l'énumération de `pointage`.

Traçabilité : les trois tables sont en insertion seule, garanties par le trigger `deny_mutation`. Modifier une excuse revient à révoquer la version en place et à en insérer une nouvelle qui la référence ; annuler revient à insérer une révocation. Rien n'est jamais effacé.

**Ce que ce dispositif ne fournit pas :** ni chaînage par hash, ni ancrage RFC 3161. Le trigger résiste à l'application et au propriétaire des tables, mais un administrateur de la base peut le désactiver ; une réécriture de l'historique des excuses serait alors **indétectable**, contrairement à une réécriture de `presence_ledger`.

Cette asymétrie est délibérée : le registre de présence doit être opposable à l'étudiant, l'historique des excuses documente une décision interne de l'établissement. Elle est signalée ici pour qu'aucune garantie ne soit supposée par analogie. Détail RGPD, dont l'absence délibérée de tout motif : [`rgpd-dpo.md`](rgpd-dpo.md) §3 bis.

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

### Vérifier un témoin externe (audit)

Page **Audit présences** du front web admin (`apps/web-admin`) : l'auditeur
colle un jeton RFC 3161 reçu par e-mail depuis la boîte externe et obtient un
verdict (`CONFORME`, `REECRITURE_DETECTEE`, `CHAINE_CORROMPUE`, …). API :

```bash
curl -X POST -H "Authorization: Bearer <token_admin>" -H "Content-Type: application/json" \
     -d '{"token": "<base64 ou PEM>", "tsa_cert": "<PEM optionnel>"}' \
     https://vecu-etudiant-admin-2.mines-ales.fr/api/v2/presence/ledger/verify-witness
```

Verdicts, principe et procédure de dichotomie : voir
[temoin-externe.md](temoin-externe.md), §6.

### Renvoyer un témoin externe en échec

```bash
curl -X POST -H "Authorization: Bearer <token_admin>" \
     https://vecu-etudiant-admin-2.mines-ales.fr/api/v2/presence/witness/resend/<anchorID>
```

### Télécharger le certificat FreeTSA

```bash
make fetch-freetsa-cert
# Vérifier l'empreinte affichée sur https://freetsa.org avant de faire confiance.
```

---

*Document — Juin 2026 — REX-IMT / Mines Alès IMT*
