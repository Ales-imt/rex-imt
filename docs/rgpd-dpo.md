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

L'authentification s'appuie sur l'annuaire **LDAP** de l'école ; aucun mot de passe n'est stocké par REX-IMT. Les référentiels étudiants/promotions/groupes/emplois du temps sont synchronisés automatiquement depuis le système d'information scolarité **Cybema/Aurega** de l'école.

Les données sensibles en transit (IP, identifiant technique) sont chiffrées par clé publique (`age`, X25519). Les présences sont protégées par un registre chaîné par hash SHA-256 et scellé périodiquement par horodatage externe (RFC 3161). Une purge RGPD automatique (anonymisation à 1 an, suppression à 3 ans) tourne toutes les 24 h.

Trois vulnérabilités connues (LDAP sans TLS, durée de refresh token, HTTP vers le SI scolarité) et une non-conformité potentielle de la purge des comptes sortis (§10.4) sont documentées ci-dessous et doivent être arbitrées avec le DPO avant présentation externe ou audit CNIL.

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
| 3 | Feedbacks pédagogiques | Recueillir les retours des étudiants sur les enseignements pour en améliorer la qualité | Mission de service public (Art. 6.1.e) | Contenu du message, pseudo choisi, promotion, groupe, IP + identifiant technique chiffrés (`strongbox`) | Étudiants | Contenu : 3 ans · données techniques (LCEN) : 1 an |
| 4 | Réponses aux feedbacks (modération) | Permettre à l'équipe pédagogique de répondre aux feedbacks | Mission de service public (Art. 6.1.e) | Contenu de la réponse, auteur (enseignant/admin) | Enseignants / admins (auteurs), étudiants (destinataires) | Alignée sur le feedback parent |
| 5 | Évaluations de cours | Mesurer la satisfaction et la charge perçue par matière, à des fins statistiques | Mission de service public (Art. 6.1.e) | Scores par dimension, pseudo, verbatims libres, IP + identifiant technique chiffrés | Étudiants (déclaratif, non lié à l'identité côté lecture) | Verbatims : anonymisés à 1 an · scores : conservés à des fins statistiques |
| 6 | Pointage de présence | Établir la preuve d'assiduité aux séances (registre d'assiduité opposable) | **Obligation légale** (Art. 6.1.c — Art. L123-1 Code de l'éducation) | `user_id`, `seance_id`, statut (présent/retard), heure de pointage, maillon de hash SHA-256, jeton d'horodatage RFC 3161 | Étudiants | Durée de la scolarité + 5 ans |
| 7 | Classification automatique des feedbacks | Catégoriser, évaluer l'urgence et résumer les feedbacks pour les équipes pédagogiques, via un modèle de langage | Mission de service public (Art. 6.1.e) — traitement instrumental du traitement n°3 | Contenu du feedback (transmis au modèle), catégorie, sous-catégorie, sentiment, urgence, résumé généré | Étudiants (auteurs des feedbacks classifiés) | Anonymisée à 1 an, supprimée à 3 ans (alignée sur le feedback) |
| 8 | Gestion des comptes / purge RGPD | Anonymiser ou supprimer les données à l'expiration des durées de conservation ; purger les comptes des étudiants sortis | Obligation légale (RGPD, minimisation) | Toutes les tables ci-dessus | Étudiants sortis | Exécution automatique toutes les 24 h (voir §7 et §10.4) |

---

## 4. Cartographie technique des données (pour l'inventaire du DPO)

| Table PostgreSQL | Contenu personnel | Chiffrement / protection |
|---|---|---|
| `user` | nom, prénom, e-mail, rôles | — |
| `student` | lien vers l'identifiant Cybema (`external_id`) | — |
| `feedback` | contenu, pseudo, `strongbox` (IP + user id chiffrés `age`) | `strongbox` chiffré par clé publique X25519 |
| `feedback_classification` | catégorisation IA du feedback (dérivée) | — |
| `eval_session` / `eval_scores` / `eval_verbatim` | pseudo, scores, verbatim libre, `strongbox` | `strongbox` chiffré par clé publique X25519 |
| `postit` / `reponse` | réponses de modération, auteur | — |
| `pointage` | présence brute (mutable) | — |
| `presence_ledger` | présence certifiée (append-only, chaînée SHA-256) | intégrité garantie par chaînage de hash |
| `presence_anchor` | jeton RFC 3161 scellant le registre | seul un hash SHA-256 quitte le système (voir §5) |
| `refresh_tokens` | jeton de session (hashé SHA-256), `user_id` | jamais stocké en clair |

Schéma complet : [`backend/schema.sql`](../backend/schema.sql).

---

## 5. Destinataires, sous-traitants et transferts

| Tiers | Rôle | Données transmises | Localisation / statut |
|---|---|---|---|
| **Annuaire LDAP** (interne école) | Authentification | Identifiants de connexion (vérification seule, rien n'est stocké côté REX-IMT) | Interne IMT Mines Alès |
| **Cybema/Aurega** (`webdfd.mines-ales.fr`) | Système de scolarité, source de vérité des référentiels étudiants/planning | Lecture seule : identités, promotions, groupes, planning | Interne IMT Mines Alès |
| **FreeTSA** (`freetsa.org`) | Autorité d'horodatage (RFC 3161) scellant le registre de présence | **Uniquement** un hash SHA-256 (aucune donnée nominative) | Tiers externe public — statut/localisation à faire valider par le DPO |
| **IA "rack"** (infrastructure IA de l'école) | Classification automatique des feedbacks | Contenu texte des feedbacks | Interne IMT Mines Alès (auto-hébergé, appel mTLS) |
| **IA "ollama" (local)** | Alternative interne pour classification (environnement de développement) | Contenu texte des feedbacks | Interne (réseau Docker local) |
| **GHCR (GitHub Container Registry)** | Hébergement des images Docker de l'application | Code applicatif uniquement — **aucune donnée personnelle** | Infrastructure GitHub |

### ⚠️ Point d'attention — fournisseur IA externe configurable ("ragarenn")

La configuration admin (`backend/admin/cmd/config.yaml`) prévoit un **troisième fournisseur IA optionnel**, `ragarenn`, pointant vers `https://ragarenn.eskemm-numerique.fr` — un service **hébergé hors de l'infrastructure de l'école**. Il n'est **pas actif en production** (le fournisseur configuré est `rack`, auto-hébergé), mais son activation enverrait le contenu texte des feedbacks à un tiers externe et **doit faire l'objet d'une analyse préalable du DPO** (nature du sous-traitant, localisation, garanties contractuelles) avant toute bascule.

Aucun autre transfert hors UE identifié dans le code.

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
| Verbatims d'évaluation | Anonymisés à 1 an | `anonymizeEvalVerbatim` | `backend/admin/pkg/rgpd/purge.go` |
| Classification IA des feedbacks | Anonymisée à 1 an, supprimée à 3 ans | `anonymizeClassification` / `deleteClassification` | `backend/admin/pkg/rgpd/purge.go` |
| Données de présence (`pointage`, `presence_ledger`, `presence_anchor`) | Scolarité + 5 ans (obligation légale) | Aucune purge automatique implémentée — cohérent avec l'obligation légale de conservation, mais la procédure d'anonymisation en fin de délai **n'est pas codée** (voir §10.5) | — |
| Tokens de session (refresh token) | 3 mois puis suppression automatique | `CleanUpTokens`, `DELETE FROM refresh_tokens WHERE expires_at < NOW()`, exécuté toutes les 12 h | `backend/common/pkg/auth/jwt_services.go` |

Purge exécutée automatiquement toutes les 24 h (`StartPurge`, `backend/admin/pkg/rgpd/purge.go`).

---

## 8. Droits des personnes concernées

| Droit | Mise en œuvre dans REX-IMT |
|---|---|
| Accès (Art. 15) | Sur demande au DPO (dpo@mines-ales.fr) — pas d'export libre-service actuellement dans l'application |
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

### 10.6 Fournisseur IA externe "ragarenn" non verrouillé
Voir §5 — configuration présente mais inactive ; risque d'activation accidentelle sans analyse préalable.

---

## 11. Documents complémentaires

- [`presence.md`](presence.md) — garanties juridiques et RGPD détaillées du registre de présence (signature électronique, eIDAS, horodatage)
- [`securite.md`](securite.md) — audit de sécurité complet et plan d'action
- [`deploiement.md`](deploiement.md) — architecture d'hébergement (VMs dédiées Mines Alès, Ansible, aucun cloud tiers pour les données)
- [`information-utilisateurs-apropos.md`](information-utilisateurs-apropos.md) — texte d'information affiché aux utilisateurs dans l'application mobile

---

*Document préparé pour la présentation au DPO — état du code au 2026-07-01 (branche `annee`).*
