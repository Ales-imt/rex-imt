# Témoin externe — archive hors base des ancrages de présence

## Objet du document

Décrit le dispositif de **témoin externe** : après chaque nouvel ancrage RFC 3161
du registre de présences, la tête de chaîne est envoyée par e-mail vers une boîte
externe. Ce document précise le rôle du témoin dans le modèle de menace, les
**exigences de déploiement non garanties par le code**, et les limites exactes de
la garantie apportée.

---

## 1. Rôle dans le modèle de menace : résistance à un initié

Le chaînage cryptographique (`presence_ledger`) et l'ancrage TSA
(`presence_anchor`) protègent contre l'altération par un attaquant externe ou un
accès direct à la base. Ils ne suffisent pas contre un **initié disposant des
accès infrastructure** : un administrateur de la base pourrait réécrire la chaîne
entière, supprimer les ancres, puis ré-ancrer la chaîne falsifiée auprès de la TSA.

Le témoin externe ferme ce scénario par la **détection** : à chaque nouvelle
ancre, un e-mail contenant `ledger_seq`, `anchored_hash`, la date d'ancrage, le
jeton RFC 3161 brut et le certificat TSA part vers une boîte que l'initié ne
contrôle pas. Une chaîne réécrite puis ré-ancrée produit de nouveaux hashes qui
**ne peuvent pas correspondre aux témoins déjà reçus**. L'e-mail bénéficie en
outre de l'horodatage propre du serveur de messagerie destinataire, indépendant
de l'infrastructure REX-IMT.

> **Le système garantit la détectabilité d'une altération, pas son
> impossibilité.** Un initié peut toujours réécrire la base ; il ne peut pas
> faire coïncider le résultat avec les témoins archivés chez un tiers.

## 2. Exigences de déploiement (impératives, non vérifiables par le code)

Le dispositif n'a de valeur anti-initié que si les deux conditions suivantes
sont réunies. **Le code ne peut pas les garantir : ce sont des exigences
organisationnelles à vérifier au déploiement.**

1. **Séparation des pouvoirs** : la boîte destinataire
   (`presence.witness.recipients`) doit être contrôlée par une personne ou un
   rôle **distinct** de ceux qui détiennent les accès base/infrastructure
   (ex. DPO, direction des études, huissier, boîte institutionnelle d'un autre
   établissement).
2. **Accès d'envoi uniquement** : le compte SMTP configuré
   (`presence.witness.smtp`) ne doit avoir **que le droit d'envoyer**. Il ne
   doit disposer d'aucun droit de lecture, de modification ou de suppression
   sur la boîte destinataire. Un initié qui pourrait purger la boîte annulerait
   le dispositif.

## 3. Contenu du témoin

Un e-mail par nouvelle ancre, contenant :

| Élément | Rôle |
|---|---|
| `ledger_seq` (corps) | Position du maillon de tête ancré |
| `anchored_hash` (corps) | Empreinte SHA-256 (64 hex) de la tête de chaîne |
| Date d'ancrage UTC + URL TSA (corps) | Contexte de vérification |
| `anchor-<seq>.tsr` (pièce jointe) | Jeton RFC 3161 brut, identique à celui archivé en base |
| `tsa-cert.pem` (pièce jointe) | Certificat TSA — vérification possible même si la TSA disparaît |

Le témoin est **autoportant** : jeton + certificat + seq + hash + date suffisent
à une vérification future sans dépendre de la TSA ni de la base.

**RGPD** : seuls un hash, un numéro de séquence, une date et une URL de TSA
sortent du système. Aucune donnée personnelle d'étudiant ne figure dans le
témoin (vérifié par test, `TestBuildWitnessMessage_NoPII`).

## 4. Limite de la garantie : la fenêtre non couverte

Le témoin prouve la conformité de la chaîne **jusqu'au maillon `seq` daté**.
Au-delà du dernier témoin reçu, seul le chaînage interne garantit l'intégrité.

La fenêtre non couverte est donc bornée par la fréquence d'ancrage :
l'ancrage automatique tourne **toutes les heures** et n'ancre (donc ne
« témoigne ») que si la tête de chaîne a bougé. Les pointages enregistrés
depuis le dernier ancrage sont protégés par le chaînage seul jusqu'au
prochain passage.

## 5. Fonctionnement technique

- Scheduler horaire (`presence.StartAnchorScheduler`, démarré dans
  `backend/admin/cmd/main.go`) : `AnchorLast` puis envoi du témoin pour chaque
  ancre **nouvellement créée**. Tête inchangée ⇒ aucune ancre ⇒ aucun e-mail.
- L'ancrage manuel (`POST /api/v2/presence/ledger/anchor`) déclenche le même envoi.
- **Un échec SMTP ne fait jamais échouer l'ancrage** : l'ancre en base reste
  valide, la tentative est tracée `FAILED` dans `presence_witness` et peut être
  rejouée.
- Idempotence : une ligne `SENT` par couple (`anchor_id`, `recipient`) dans
  `presence_witness` ; un témoin déjà envoyé n'est jamais ré-émis. Cette table
  indique aussi **jusqu'où** le témoin externe fait foi.
- Si `presence.witness.enabled` est faux ou le SMTP non configuré, l'ancrage
  fonctionne normalement et l'envoi est simplement ignoré (log d'information).

### Renvoyer un témoin en échec

```bash
curl -X POST -H "Authorization: Bearer <token_admin>" \
     https://vecu-etudiant-admin-2.mines-ales.fr/api/v2/presence/witness/resend/<anchorID>
```

Réponse : `{"anchorId": N, "sent": n, "skipped": n, "failed": n}` — `skipped`
compte les destinataires déjà servis (idempotence).

### Configuration

```yaml
presence:
  witness:
    enabled: true
    recipients:
      - archive-presences@exemple-externe.org   # boîte contrôlée par un TIERS
    smtp:
      host: ${SMTP_HOST}
      port: 587
      username: ${SMTP_USERNAME}
      password: ${SMTP_PASSWORD}
      from: rex-imt-noreply@mines-ales.fr
      startTLS: true
      timeout: 10s
```

Les secrets passent par variables d'environnement (`${…}` substitué au
chargement de la config). Aucun secret en dur.

## 6. Vérifier un témoin : la page de vérification

Page **Audit présences** du front web admin (`apps/web-admin`, segment
`presence_witness_workflow`, rôles admin/gestionnaire), adossée à deux
endpoints du backend admin :

```
POST /api/v2/presence/ledger/verify-witness   # {token, tsa_cert} → verdict JSON
GET  /api/v2/presence/ledger/anchors          # repères : ancres en base (seq, date, TSA)
```

L'auditeur y colle **un seul témoin** : le jeton RFC 3161 reçu par e-mail
(pièce jointe `anchor-<seq>.tsr`, collée en base64/PEM ou téléversée telle
quelle), éventuellement accompagné du certificat TSA (`tsa-cert.pem`). À défaut
de certificat collé, le CA racine configuré (`presence.timestamp.caCertPath`)
sert de point de confiance.

> **Provenance du témoin** : la preuve vient de la confrontation de **deux
> sources indépendantes** — le témoin détenu par le tiers et la base. Il faut
> coller le jeton **reçu par e-mail**, jamais un jeton rechargé depuis la base
> (qui ne prouverait que la cohérence de la base avec elle-même).

### Principe : pourquoi un seul témoin suffit

1. Le jeton, signature CMS et chaîne de certification validées, prouve qu'un
   **hash de tête H existait à une date T** certifiée par la TSA.
2. Si H figure encore dans `presence_ledger.hash` (maillon N), la chaîne
   actuelle reproduit l'état scellé ; sinon, elle a été réécrite après T.
3. `VerifyChain` confirme en complément la cohérence interne du chaînage
   jusqu'à N.

### Sens des verdicts

| Verdict | Signification |
|---|---|
| `CONFORME` | Chaîne conforme jusqu'au maillon N, scellé le T par la TSA. Tout ce qui précède ce point est authentifié. |
| `REECRITURE_DETECTEE` | Le hash certifié le T est **absent** de la chaîne actuelle : les données ont été modifiées après cette date (altération d'un maillon ≤ N). |
| `CHAINE_CORROMPUE` | Le hash scellé existe encore, mais le chaînage interne est rompu **avant** lui (le `seq` de rupture est affiché). |
| `TOKEN_INVALIDE` | Jeton illisible (copier-coller incomplet, mauvais fichier). Témoin non exploitable. |
| `SIGNATURE_INVALIDE` | La chaîne de certification TSA ne remonte pas au certificat de confiance : témoin **non probant**. |

La vérification est **en lecture seule** : aucun témoin collé n'est conservé,
rien n'est écrit en base.

### Localiser une altération par dichotomie (manuelle)

La page traite volontairement un témoin à la fois ; la dichotomie est pilotée
par l'auditeur, qui détient les témoins externes :

1. tester le témoin le plus récent ; conforme ⇒ chaîne intègre jusqu'à sa date ;
2. verdict négatif ⇒ tester un témoin plus ancien : s'il est conforme,
   l'altération se situe **entre les deux dates certifiées** ;
3. répéter en resserrant l'intervalle. La date certifiée (`sealed_at`) affichée
   à chaque test situe le témoin sur la frise temporelle.

La page liste aussi, à titre de **repère seulement**, les dates/`seq` des ancres
présentes en base : ces dates viennent de la base et ne constituent pas une
preuve — la preuve reste les témoins externes collés par l'auditeur.

### Limite (rappel)

Le dispositif garantit la **détectabilité** d'une altération, pas son
impossibilité, et cette garantie repose sur des témoins **externes** archivés
chez un tiers. Un verdict `CONFORME` ne couvre que ce qui précède le maillon
scellé : la fenêtre postérieure au dernier témoin reste protégée par le seul
chaînage interne (§4).

---

*Document — Juillet 2026 — REX-IMT / Mines Alès IMT*
