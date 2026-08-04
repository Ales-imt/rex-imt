# REX-IMT — Documentation RGPD à destination du DPO

## Objet du document

Ce document rassemble les éléments nécessaires à la présentation de REX-IMT au délégué à la protection des données (DPO) d'IMT Mines Alès : registre des traitements, cartographie des données, sous-traitants, mesures de sécurité, durées de conservation, droits des personnes, et points de vigilance identifiés dans le code qui nécessitent un arbitrage du DPO.

Il complète deux documents techniques existants :
- [`securite.md`](securite.md) — audit de sécurité détaillé
- [`presence.md`](presence.md) — valeur juridique et RGPD du registre de présence (chaînage cryptographique, horodatage RFC 3161)

Une copie du texte affiché aux utilisateurs (écran « À propos ») figure dans [`information-utilisateurs-apropos.md`](information-utilisateurs-apropos.md).

---

## 1. Résumé exécutif

REX-IMT est une plateforme pédagogique composée de trois applications (admin web, élève web, élève mobile) utilisée par IMT Mines Alès pour :

- collecter des **feedbacks** et **évaluations de cours** auprès des étudiants,
- **pointer la présence** des étudiants en séance (obligation légale d'assiduité),
- assister les équipes pédagogiques via une **classification automatique par IA** des feedbacks.

Tout contenu textuel libre écrit par un étudiant — feedback comme verbatim d'évaluation — passe par une **modération humaine préalable** : il n'est ni diffusé à l'équipe pédagogique, ni transmis au modèle de langage, tant qu'un modérateur ne l'a pas publié. Un contenu refusé est chiffré au repos et détruit sous 90 jours (voir §3 traitement n°9 et §7).

L'authentification s'appuie sur l'annuaire **LDAP** de l'école ; aucun mot de passe n'est stocké par REX-IMT. Les référentiels étudiants/promotions/groupes/emplois du temps sont synchronisés automatiquement depuis le système d'information scolarité **Cybema/Aurega** de l'école.

Les données sensibles en transit (IP, identifiant technique) sont chiffrées par clé publique (`age`, X25519). Les présences sont protégées par un registre chaîné par hash SHA-256 et scellé périodiquement par horodatage externe (RFC 3161). Une purge RGPD automatique (anonymisation à 1 an, suppression à 3 ans) tourne toutes les 24 h.

Trois vulnérabilités connues (LDAP sans TLS, durée de refresh token, HTTP vers le SI scolarité) et une non-conformité potentielle de la purge des comptes sortis (§10.4) sont documentées ci-dessous et doivent être arbitrées avec le DPO avant présentation externe ou audit CNIL. Le pipeline de modération, plus récent, appelle trois arbitrages complémentaires (§10.7 à §10.9) : périmètre du rôle modérateur, exposition de la promotion dans la file, et conservation de 90 jours des contenus refusés.

Le point d'attention signalé dans la révision précédente sur le fournisseur IA externe « ragarenn » est **résolu** : l'adresse hors infrastructure école a été retirée de la configuration, aucun transfert vers un tiers n'est atteignable par paramétrage (§5 et §10.6).

---

## 2. Responsable de traitement et acteurs

| Rôle | Identité |
|---|---|
| **Responsable de traitement** | IMT Mines Alès — 6 avenue de Clavières, 30319 Alès Cedex |
| **DPO** | dpo@mines-ales.fr |
| **Personnes concernées** | Étudiants (utilisateurs principaux), enseignants et personnel administratif (comptes admin, auteurs de réponses aux feedbacks) |
| **Éditeur / mainteneur technique** | Équipe projet REX-IMT (interne IMT Mines Alès) — dépôt `github.com/Ales-imt/rex-imt`, licence AGPL v3 |

---

## 3. Registre des activités de traitement (Art. 30 RGPD)

| # | Traitement | Finalité | Base légale | Données traitées | Personnes concernées | Durée de conservation |
|---|---|---|---|---|---|---|
| 1 | Authentification | Identifier l'utilisateur et sécuriser l'accès à l'application | Mission de service public (Art. 6.1.e) | Login/mot de passe LDAP (non stockés — vérifiés par l'annuaire école), JWT de session | Étudiants, enseignants, admins | JWT : 15 min · refresh token : 90 jours (voir §7) |
| 2 | Synchronisation de l'annuaire scolarité | Maintenir à jour la liste des étudiants, promotions, groupes, emplois du temps | Mission de service public (Art. 6.1.e) | Nom, prénom, e-mail, promotion, groupe, matières, séances (source : Cybema/Aurega, système de scolarité de l'école) | Étudiants, enseignants | Alignée sur le statut scolarité (voir purge §7) |
| 3 | Feedbacks pédagogiques | Recueillir les retours des étudiants sur les enseignements pour en améliorer la qualité | Mission de service public (Art. 6.1.e) | Contenu du message (`raw_content` avant relecture, `content` après publication), pseudo choisi, promotion, groupe, IP + identifiant technique chiffrés (`strongbox`) | Étudiants | Contenu publié : 3 ans · contenu refusé : 90 jours · données techniques (LCEN) : 1 an |
| 4 | Réponses aux feedbacks | Permettre à l'équipe pédagogique de répondre aux feedbacks publiés | Mission de service public (Art. 6.1.e) | Contenu de la réponse, auteur (enseignant/admin) | Enseignants / admins (auteurs), étudiants (destinataires) | Alignée sur le feedback parent |
| 5 | Évaluations de cours | Mesurer la satisfaction et la charge perçue par matière, à des fins statistiques | Mission de service public (Art. 6.1.e) | Scores par dimension, pseudo, verbatims libres (`raw_texte` avant relecture, `texte` après publication), IP + identifiant technique chiffrés | Étudiants (déclaratif, non lié à l'identité côté lecture) | Verbatims publiés : anonymisés à 1 an · verbatims refusés : 90 jours · scores : conservés à des fins statistiques |
| 6 | Pointage de présence | Établir la preuve d'assiduité aux séances (registre d'assiduité opposable) | **Obligation légale** (Art. 6.1.c — Art. L123-1 Code de l'éducation) | `user_id`, `seance_id`, statut (présent/retard), heure de pointage, maillon de hash SHA-256, jeton d'horodatage RFC 3161 | Étudiants | Durée de la scolarité + 5 ans |
| 7 | Classification automatique des feedbacks | Catégoriser, évaluer l'urgence et résumer les feedbacks pour les équipes pédagogiques, via un modèle de langage | Mission de service public (Art. 6.1.e) — traitement instrumental du traitement n°3 | Contenu du feedback **publié uniquement** (transmis au modèle), catégorie, sous-catégorie, sentiment, urgence, résumé généré | Étudiants (auteurs des feedbacks classifiés) | Anonymisée à 1 an, supprimée à 3 ans (alignée sur le feedback) |
| 8 | Gestion des comptes / purge RGPD | Anonymiser ou supprimer les données à l'expiration des durées de conservation ; purger les comptes des étudiants sortis | Obligation légale (RGPD, minimisation) | Toutes les tables ci-dessus | Étudiants sortis | Exécution automatique toutes les 24 h (voir §7 et §10.4) |
| 9 | **Modération préalable des contenus libres** | Relire tout texte libre écrit par un étudiant (feedback, verbatim d'évaluation) avant sa diffusion à l'équipe pédagogique et avant tout traitement par IA | Mission de service public (Art. 6.1.e) — traitement instrumental des traitements n°3 et n°5 ; contribue également au respect de l'Art. 6-I LCEN | Texte brut à relire, horodatage, promotion de rattachement, dimension (verbatims) ; identité du modérateur (`moderated_by`), date de décision, motif de refus | Étudiants (auteurs), modérateurs (traçabilité de la décision) | Décision et motif alignés sur le contenu parent · texte refusé : 90 jours (chiffré) |

---

## 4. Cartographie technique des données (pour l'inventaire du DPO)

| Table PostgreSQL | Contenu personnel | Chiffrement / protection |
|---|---|---|
| `user` | nom, prénom, e-mail, rôles | — |
| `student` | lien vers l'identifiant Cybema (`external_id`) | — |
| `feedback` | contenu publié (`content`), texte brut en attente de relecture (`raw_content`), pseudo, `strongbox` (IP + user id chiffrés `age`), état de modération (`moderation_status`, `moderated_by`, `moderated_at`, `rejection_reason`) | `strongbox` chiffré par clé publique X25519 · `raw_content` chiffré `age` dès qu'un contenu est refusé |
| `feedback_classification` | catégorisation IA du feedback (dérivée) | alimentée uniquement à partir de contenus publiés |
| `eval_session` / `eval_scores` | pseudo, scores | — |
| `eval_verbatim` | verbatim publié (`texte`), texte brut en attente de relecture (`raw_texte`), `strongbox`, état de modération (mêmes colonnes que `feedback`) | `strongbox` chiffré par clé publique X25519 · `raw_texte` chiffré `age` dès qu'un verbatim est refusé |
| `postit` / `reponse` | réponses de modération, auteur | — |
| `pointage` | présence brute (mutable) | — |
| `presence_ledger` | présence certifiée (append-only, chaînée SHA-256) | intégrité garantie par chaînage de hash |
| `presence_anchor` | jeton RFC 3161 scellant le registre | seul un hash SHA-256 quitte le système (voir §5) |
| `refresh_tokens` | jeton de session (hashé SHA-256), `user_id` | jamais stocké en clair |

Schéma complet : [`backend/schema.sql`](../backend/schema.sql).

---

## 4 bis. Pipeline de modération préalable (traitement n°9)

Depuis les changesets Liquibase `019` (feedbacks) et `020` (verbatims d'évaluation), **aucun texte libre écrit par un étudiant n'est diffusé ni analysé avant relecture humaine**. Le mécanisme est identique pour les deux types de contenu.

### Cycle de vie d'un contenu

| État | Où est le texte | Qui y a accès | Diffusion / IA |
|---|---|---|---|
| `PENDING` (par défaut à la création) | `raw_content` / `raw_texte`, en clair | Modérateurs uniquement | Aucune |
| `PUBLISHED` | `content` / `texte` ; le texte brut est **effacé** dans le même UPDATE | Équipe pédagogique, IA | Oui |
| `REJECTED` | `raw_content` / `raw_texte` **remplacé par sa version chiffrée `age`** ; le champ publié reste NULL | Personne (illisible au repos, y compris par les administrateurs) | Jamais |

Le modérateur peut corriger le texte avant publication ; c'est la version corrigée qui est diffusée.

### Garanties techniques vérifiables

- **L'IA ne reçoit que du contenu publié.** Pour les feedbacks, le trigger `feedback_notify` a été déplacé de l'`INSERT` vers la transition vers `PUBLISHED` (changeset `019b`). Pour les verbatims, la requête alimentant le prompt filtre `moderation_status = 'PUBLISHED'` (`GetVerbatimsForPrompt`).
- **Le contenu refusé ne quitte jamais le serveur.** Les endpoints de consultation côté étudiant substituent un texte de remplacement au blob chiffré (`GetChatHistoryByPseudo`, `GetVerbatimsBySessionAndPseudo`).
- **La file de modération est minimisée.** Elle expose le texte à relire, son horodatage, sa promotion de rattachement et, pour les verbatims, la dimension évaluée. Elle n'expose ni `strongbox`, ni le pseudo (choisi librement, il peut contenir un nom), ni `session_id` (qui permettrait de remonter à l'auteur via `eval_session`). Pour les verbatims, la promotion est déduite du **cours évalué** (`matiere → periode → promotion`), jamais de l'auteur.
- **Accès restreint.** Les routes `/moderation` sont réservées aux rôles `ADMIN` et `MODERATEUR` — le rôle `GESTIONNAIRE`, qui accède aux feedbacks publiés, n'y a pas accès (voir §10.7).
- **Décision tracée.** `moderated_by` et `moderated_at` conservent l'auteur et la date de chaque décision ; `rejection_reason` porte le motif communiqué à l'étudiant.

### Information de la personne concernée

L'étudiant suit l'état de ses propres contenus dans l'application, sans qu'aucun lien auteur→contenu ne soit reconstruit côté serveur : la preuve de possession repose sur le pseudo qu'il fournit (et, pour les feedbacks, sur les `message_id` qu'il détient). Il voit l'état *en attente*, *publié* ou *refusé avec son motif*.

Sources : `backend/admin/pkg/moderation/`, `infras/liquibase/releases/v0.0.1/019-*.yaml` et `020-*.yaml`.

---

## 5. Destinataires, sous-traitants et transferts

| Tiers | Rôle | Données transmises | Localisation / statut |
|---|---|---|---|
| **Annuaire LDAP** (interne école) | Authentification | Identifiants de connexion (vérification seule, rien n'est stocké côté REX-IMT) | Interne IMT Mines Alès |
| **Cybema/Aurega** (`webdfd.mines-ales.fr`) | Système de scolarité, source de vérité des référentiels étudiants/planning | Lecture seule : identités, promotions, groupes, planning | Interne IMT Mines Alès |
| **FreeTSA** (`freetsa.org`) | Autorité d'horodatage (RFC 3161) scellant le registre de présence | **Uniquement** un hash SHA-256 (aucune donnée nominative) | Tiers externe public — statut/localisation à faire valider par le DPO |
| **IA "rack"** (infrastructure IA de l'école) | Classification automatique des feedbacks, synthèse des évaluations | Contenu texte **publié uniquement** (feedbacks et verbatims relus par un modérateur — voir §4 bis) | Interne IMT Mines Alès (auto-hébergé, appel mTLS) |
| **IA "ollama" (local)** | Alternative interne pour classification (environnement de développement) | Idem — contenu publié uniquement | Interne (réseau Docker local) |
| **GHCR (GitHub Container Registry)** | Hébergement des images Docker de l'application | Code applicatif uniquement — **aucune donnée personnelle** | Infrastructure GitHub |

### ✅ Résolu — retrait du fournisseur IA externe ("ragarenn")

La révision précédente de ce document (commit `dd77a45`) signalait un **troisième fournisseur IA optionnel**, `ragarenn`, pointant vers `https://ragarenn.eskemm-numerique.fr`, service hébergé **hors de l'infrastructure de l'école**.

**Cette configuration a été retirée.** L'adresse externe n'existe plus dans le dépôt : le bloc `ragarenn:` a disparu des deux fichiers de configuration (`backend/admin/cmd/config.yaml` et `infras/run/config-admin.yaml`) au commit `cf073a3` (2026-07-31), et aucune occurrence du domaine `eskemm-numerique.fr` ne subsiste nulle part dans le code, l'infrastructure ou les playbooks Ansible.

**Conséquence pratique :** sélectionner `provider: ragarenn` aujourd'hui produirait une `baseURL` vide — le connecteur échouerait au premier appel sans joindre aucun tiers. Le rétablissement d'un transfert externe supposerait de **réintroduire délibérément une adresse** dans la configuration, ce qui ne peut pas résulter d'une simple bascule de paramètre.

**Résidu de code, sans effet sur les données** (voir §10.6) : le nom du fournisseur reste sélectionnable dans le code — package `backend/admin/pkg/ia/ragarenn/`, structure `RAGaRennConfig` (`backend/common/pkg/services/config.go`), deux `case "ragarenn"` dans `backend/admin/cmd/main.go`, mention dans le commentaire `# "ollama" ou "ragarenn" ou "rack"` des fichiers de configuration, et variable `RAGARENN_API_KEY` dans `.vscode/secrets.env.example`.

Le fournisseur actif en production reste `rack`, auto-hébergé sur l'infrastructure de l'école.

Aucun transfert hors UE identifié dans le code.

---

## 6. Mesures de sécurité (synthèse)

Détail complet et plan d'action dans [`securite.md`](securite.md). Score de l'audit : **8.4/10**.

**Mesures en place :**
- Authentification déléguée au LDAP — aucun mot de passe stocké
- Refresh tokens hashés en base (jamais en clair)
- Chiffrement asymétrique (`age`, X25519) de l'IP et de l'identifiant technique dans les feedbacks/évaluations
- Registre de présence inaltérable (chaînage SHA-256 + horodatage RFC 3161)
- Secrets de production chiffrés (Ansible Vault)
- TLS en transit sur les applications (nginx, certificats gérés)
- Headers de sécurité HTTP (CSP, HSTS, X-Frame-Options, etc.)
- Injection LDAP impossible (`ldap.EscapeFilter()` systématique)
- Purge RGPD automatique toutes les 24 h

**Vulnérabilités ouvertes (au 2026-07-01, non corrigées) :**

| Sévérité | Constat | Impact RGPD |
|---|---|---|
| 🟠 Élevée | Connexion LDAP en clair (`ldap://`, pas de TLS) | Identifiants transitant en clair sur le réseau interne lors de chaque authentification |
| 🟠 Élevée | Refresh token valide 90 jours | Fenêtre d'exploitation prolongée en cas de vol de token |
| 🟡 Moyenne | Appels HTTP (non chiffrés) vers Cybema/Aurega (`webdfd.mines-ales.fr`) | Noms, emplois du temps, groupes, notes des étudiants transitant en clair sur le réseau |

---

## 7. Durées de conservation (tableau consolidé — vérifié dans le code)

| Donnée | Durée annoncée aux utilisateurs | Mécanisme technique | Vérifié dans |
|---|---|---|---|
| Compte utilisateur | Scolarité + 1 an | `purgeComptesSortis` : suppression si absent des promotions actives depuis > 1 an (requête Aurega `HAVING MAX(datefin) < NOW() - INTERVAL 1 YEAR`) | `backend/admin/pkg/rgpd/mariadb/query.sql`, `purge.go` — **voir réserve §10.4** |
| Contenu des feedbacks | 3 ans à compter de la publication | `deleteFeedback` (seuil `deleteAfter = 3` ans) | `backend/admin/pkg/rgpd/purge.go` |
| Données techniques feedback (IP/id chiffrés — LCEN) | 1 an à compter de la publication | `anonymizeFeedback` (seuil `anonymizeAfter = 1` an, met `strongbox` à NULL) | `backend/admin/pkg/rgpd/purge.go` |
| **Feedbacks refusés en modération** | 90 jours à compter de la décision de refus | `purgeRejectedFeedback` (seuil `rejectedAfter = 90` jours, sur `moderated_at`) — suppression de la ligne entière, texte chiffré et `strongbox` compris | `backend/admin/pkg/rgpd/purge.go` |
| **Verbatims refusés en modération** | 90 jours à compter de la décision de refus | `purgeRejectedVerbatim` (même seuil et même logique) | `backend/admin/pkg/rgpd/purge.go` |
| Verbatims d'évaluation publiés | Anonymisés à 1 an | `anonymizeEvalVerbatim` (met `strongbox` à NULL) | `backend/admin/pkg/rgpd/purge.go` |
| Classification IA des feedbacks | Anonymisée à 1 an, supprimée à 3 ans | `anonymizeClassification` / `deleteClassification` | `backend/admin/pkg/rgpd/purge.go` |
| Données de présence (`pointage`, `presence_ledger`, `presence_anchor`) | Scolarité + 5 ans (obligation légale) | Aucune purge automatique implémentée — cohérent avec l'obligation légale de conservation, mais la procédure d'anonymisation en fin de délai **n'est pas codée** (voir §10.5) | — |
| Tokens de session (refresh token) | 3 mois puis suppression automatique | `CleanUpTokens`, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`, exécuté toutes les 12 h | `backend/common/pkg/auth/jwt_services.go` |

Purge exécutée automatiquement toutes les 24 h (`StartPurge`, `backend/admin/pkg/rgpd/purge.go`).

**Justification du délai de 90 jours sur les contenus refusés :** un contenu refusé n'a jamais été publié, aucune obligation de conservation LCEN ne s'y applique. Le délai couvre uniquement une éventuelle contestation de la décision de modération par son auteur. Pendant cette fenêtre, le texte est conservé chiffré (`age`) et n'est lisible par personne — pas même par les administrateurs — sans la clé privée détenue hors application.

---

## 8. Droits des personnes concernées

| Droit | Mise en œuvre dans REX-IMT |
|---|---|
| Accès (Art. 15) | Sur demande au DPO (dpo@mines-ales.fr) — pas d'export libre-service actuellement dans l'application. L'étudiant consulte toutefois en autonomie ses feedbacks, ses évaluations et **l'état de modération de chacun de ses textes libres** (en attente / publié / refusé avec motif) |
| Rectification (Art. 16) | Via le SI scolarité (Cybema/Aurega) pour l'identité, source resynchronisée automatiquement ; sur demande DPO pour le reste |
| Effacement (Art. 17) | Applicable au contenu des feedbacks (suppression = anonymisation immédiate de l'affichage, purge technique différée par obligation LCEN) ; **non applicable** aux données de présence et aux données techniques pendant la durée légale de conservation (Art. 17.3.b/c RGPD) |
| Portabilité (Art. 20) | Sur demande au DPO — pas d'export structuré automatisé actuellement |
| Opposition (Art. 21) | Sur demande au DPO ; non applicable aux traitements fondés sur une obligation légale (présence) |
| Information préalable | Écran « À propos » affiché à la première connexion (voir [`information-utilisateurs-apropos.md`](information-utilisateurs-apropos.md)) ; horodatage de la prise de connaissance enregistré dans `user.informed_at` (endpoint `PATCH /me/informed`) |

**Constat :** les droits d'accès, de rectification et de portabilité reposent aujourd'hui sur une procédure manuelle (contact DPO), sans self-service dans l'application. À évaluer avec le DPO selon le volume de demandes attendu.

---

## 9. Nécessité d'une analyse d'impact (AIPD/PIA) — éléments d'appréciation

Une décision formelle appartient au DPO ; les éléments suivants visent à l'objectiver :

**Facteurs réduisant le risque :**
- Le pointage de présence répond à une **obligation légale** (Art. L123-1 Code de l'éducation), avec une finalité strictement limitée à l'assiduité.
- Les évaluations de cours sont **déclarativement anonymes** côté lecture (pas de lien direct affiché avec l'identité).
- Les données techniques sensibles (IP, identifiant) sont **chiffrées** dès la collecte (`strongbox`).
- Pas de transfert hors UE actif, pas de profilage à visée décisionnelle sur les personnes.
- **Une relecture humaine précède systématiquement tout traitement par IA** (§4 bis) : le modèle de langage ne reçoit aucun contenu brut non filtré, ce qui limite mécaniquement le risque de transmission de données non pertinentes, nominatives ou sensibles saisies par erreur dans un champ libre.

**Facteurs justifiant un examen attentif :**
- Traitement de données par **intelligence artificielle** (classification automatique du contenu des feedbacks), même à finalité d'agrégation/priorisation, relève d'une vigilance renforcée (recommandations CNIL sur l'IA).
- Le pointage de présence constitue un **suivi individuel et systématique** des étudiants sur toute la scolarité, avec conservation longue (scolarité + 5 ans).
- Combinaison de plusieurs traitements sur la même population (feedback + évaluation + présence) pouvant permettre un croisement de profils si les identifiants ne sont pas cloisonnés.

**Recommandation :** documenter formellement l'analyse (AIPD complète ou note de qualification) avec le DPO, notamment sur le volet « classification IA des feedbacks », qui est le traitement le plus récent et le moins couvert par la base légale « obligation légale » directe.

---

## 10. Points de vigilance à trancher avec le DPO

### 10.1 LDAP sans TLS
Identifiants transmis en clair sur le réseau interne à chaque authentification. Cf. `securite.md` HIGH-1.

### 10.2 Refresh token valable 90 jours
Fenêtre d'exploitation longue en cas de vol de token. Cf. `securite.md` HIGH-2.

### 10.3 Flux HTTP non chiffrés vers Cybema/Aurega
Données étudiantes (identités, planning, notes) en clair sur le réseau lors de la synchronisation. Cf. `securite.md` MED-1.

### 10.4 Purge des comptes sortis potentiellement inopérante ⚠️

**Constat technique (non documenté ailleurs) :** `purgeComptesSortis` (`backend/admin/pkg/rgpd/purge.go`) exécute `DELETE FROM "user" WHERE id = $1` (suppression physique, pas d'anonymisation). Or la table `presence_ledger` porte une contrainte `FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE RESTRICT` (`backend/schema.sql`).

**Conséquence :** dès qu'un étudiant a été pointé au moins une fois en présence, la suppression de son compte échoue silencieusement (l'erreur est journalisée mais le traitement continue sur le compte suivant). Le compte **n'est donc jamais purgé** malgré l'annonce « Durée de la scolarité + 1 an » faite aux utilisateurs — en pratique, quasiment tous les étudiants ayant suivi au moins une séance de cours sont concernés.

**Recommandation :** implémenter le mécanisme d'anonymisation décrit dans `presence.md` §4.5 (« anonymiser les données nominatives dans `users` et `pointage` sans supprimer les maillons du ledger ») au lieu d'une suppression physique bloquée par la contrainte, ou capturer l'erreur `RESTRICT` et basculer automatiquement vers une anonymisation. **À corriger avant présentation au DPO comme conforme, ou à annoncer explicitement comme écart en cours de résolution.**

### 10.5 Absence de procédure codée pour l'anonymisation post-5-ans du registre de présence
`presence.md` décrit la procédure cible (anonymiser `user`/`pointage`, conserver les hash) mais aucun job n'implémente ce cycle à ce jour — seule la conservation initiale (scolarité + 5 ans) est respectée par absence de suppression. À planifier avant que les premières échéances de 5 ans n'arrivent.

### 10.6 Fournisseur IA externe "ragarenn" — ✅ résolu, reste un nettoyage de code

**Statut : clos pour ce qui relève de l'arbitrage DPO.** L'adresse externe a été retirée de la configuration (voir §5) ; plus aucun transfert vers un tiers hors infrastructure école n'est atteignable par simple paramétrage.

Subsiste un **résidu de code sans effet sur les données** : le nom `ragarenn` reste une valeur acceptée par le sélecteur de fournisseur, avec un package et une structure de configuration désormais orphelins. Sélectionner cette valeur aboutit à une `baseURL` vide et à un échec au premier appel.

**Recommandation (hygiène, non bloquante) :** supprimer le package `backend/admin/pkg/ia/ragarenn/`, la structure `RAGaRennConfig`, les deux `case "ragarenn"` de `main.go`, la variable `RAGARENN_API_KEY` de `.vscode/secrets.env.example` et les mentions résiduelles dans les commentaires. Cela évite qu'une future réintroduction d'URL passe pour une remise en service d'un chemin déjà validé, alors qu'elle constituerait un nouveau transfert à analyser.

### 10.7 Périmètre du rôle MODERATEUR à valider

Les routes `/moderation` sont ouvertes aux rôles `ADMIN` et `MODERATEUR` (`backend/admin/cmd/main.go`). Le rôle `GESTIONNAIRE` en est volontairement exclu, alors qu'il accède par ailleurs aux feedbacks **publiés** avec leur promotion, leur groupe, leur pseudo et leur `strongbox`.

Ce cloisonnement est un choix d'implémentation, pas une exigence dérivée d'une analyse formelle. **À arbitrer avec le DPO :** qui doit être habilité à lire un contenu *avant* publication — c'est-à-dire un texte que son auteur n'a pas encore vu diffusé, et qui peut contenir des propos qu'il regrette ou des données qu'il n'aurait pas dû saisir. La désignation nominative des modérateurs et leur information sur la nature de l'accès relèvent de la même décision.

### 10.8 Exposition de la promotion dans la file de modération

Pour permettre au modérateur de filtrer sa file, chaque contenu en attente affiche sa **promotion** de rattachement. Une promotion désigne une cohorte entière (plusieurs dizaines d'étudiants) et ne permet pas d'identifier un auteur, mais elle **réduit l'ensemble d'anonymat** par rapport à l'état antérieur, où la file n'exposait que le texte et son horodatage.

Le niveau d'exposition reste très inférieur à celui déjà consenti aux gestionnaires sur les feedbacks publiés (promotion **et** groupe **et** pseudo **et** `strongbox`). **À confirmer par le DPO** comme proportionné à la finalité d'organisation du travail de modération.

### 10.9 Conservation de 90 jours des contenus refusés

Le délai de 90 jours retenu pour les contenus refusés (§7) n'est adossé à aucune obligation légale : il est justifié par la possibilité d'une contestation de la décision de modération. **À valider par le DPO**, tant sur le principe de cette conservation que sur la durée retenue. Une réduction du délai serait un simple changement de la constante `rejectedAfter` (`backend/admin/pkg/rgpd/purge.go`).

Point connexe : aucune procédure de contestation n'est aujourd'hui outillée dans l'application — l'étudiant voit le motif du refus mais ne dispose d'aucun canal de recours intégré. Si la conservation est justifiée par le recours, l'absence de canal de recours affaiblit cette justification.

---

## 11. Documents complémentaires

- [`presence.md`](presence.md) — garanties juridiques et RGPD détaillées du registre de présence (signature électronique, eIDAS, horodatage)
- [`securite.md`](securite.md) — audit de sécurité complet et plan d'action
- [`deploiement.md`](deploiement.md) — architecture d'hébergement (VMs dédiées Mines Alès, Ansible, aucun cloud tiers pour les données)
- [`information-utilisateurs-apropos.md`](information-utilisateurs-apropos.md) — texte d'information affiché aux utilisateurs dans l'application mobile

---

## Version de référence de ce document

| | |
|---|---|
| **Branche** | `main` |
| **Commit décrit** | `0dde667` — « moderation feedback et verbatin » (2026-08-03) |
| **Révision précédente** | `dd77a45` — « creation annee » (2026-07-02), antérieure au pipeline de modération préalable (§4 bis) |

Les constats de ce document ont été vérifiés dans le code à ce commit. Toute modification ultérieure du pipeline de modération, des durées de conservation (§7) ou de la configuration des fournisseurs IA (§5) invalide potentiellement les affirmations ci-dessus et doit donner lieu à une nouvelle revue.

Pour retrouver l'état exact du code décrit ici :

```bash
git show 0dde667 --stat
git diff dd77a45..0dde667 -- backend/ infras/liquibase/
```

---

*Document préparé pour la présentation au DPO — IMT Mines Alès.*
