# Texte d'information affiché aux utilisateurs — écran « À propos »

## Origine et statut de ce document

Ce fichier reproduit fidèlement, en Markdown, le texte affiché aux étudiants dans l'application mobile REX-IMT, sur l'écran « À propos » (`apps/mobile/components/apropos-content.tsx`). Il est présenté :

- **lors de la première connexion**, avant tout usage de l'application (écran `apropos-first-login.tsx`, validation explicite requise via le bouton « J'ai compris, continuer », qui déclenche `PATCH /me/informed` et horodate la prise de connaissance dans `user.informed_at`) ;
- **à tout moment ensuite**, depuis le menu (écran `apropos.tsx`).

**Ce document est une copie de travail destinée à la revue par le DPO.** Le texte de référence, juridiquement opposable, reste le composant source `apps/mobile/components/apropos-content.tsx`. Toute modification validée par le DPO doit être répercutée dans ce fichier source, pas seulement ici.

Les constats de cohérence entre ce texte et le comportement réel du code (durées de conservation, mécanismes de purge) sont documentés dans [`rgpd-dpo.md`](rgpd-dpo.md) — voir en particulier le §10.4 concernant un écart identifié sur la purge des comptes.

---

## rex-imt

*Retours d'expérience — IMT Mines Alès*

---

## Responsable de traitement

| | |
|---|---|
| Établissement | IMT Mines Alès |
| Adresse | 6 avenue de Clavières, 30319 Alès Cedex |
| Contact DPO | dpo@mines-ales.fr |

---

## Données collectées et finalités

Cette application collecte les données suivantes dans le cadre de sa mission pédagogique :

**Identité** — Nom, prénom, adresse e-mail institutionnelle — utilisés pour l'authentification via le LDAP de l'école.

**Feedbacks** — Contenu textuel, pseudo choisi, promotion et groupe — collectés pour améliorer la qualité des enseignements. Chaque message est relu par un modérateur avant toute diffusion ou analyse (voir ci-dessous).

**Évaluations** — Scores par dimension pédagogique et verbatims optionnels. Comme les feedbacks, chaque verbatim est relu par un modérateur avant toute diffusion ou analyse, et l'évaluation est associée à une donnée d'identification chiffrée (adresse IP et identifiant technique) conservée pour répondre aux obligations légales ; elle n'est ni affichée ni accessible à l'équipe pédagogique.

**Présences** — Identifiant de séance, statut (présent / retard) et heure de pointage — collectés par scan de QR code lors des séances. Un registre cryptographique d'intégrité garantit l'inaltérabilité de ces données. Base légale : obligation légale d'assiduité (Art. L123-1 Code de l'éducation).

**Données techniques** — Adresse IP et identifiant technique chiffrés — conservés pour répondre aux obligations légales (voir ci-dessous).

> Base légale principale : mission de service public d'enseignement supérieur (Art. 6.1.e RGPD). Pour les données de présence : obligation légale d'assiduité (Art. 6.1.c RGPD — Art. L123-1 Code de l'éducation).

---

## Durées de conservation

| Donnée | Durée |
|---|---|
| Votre compte | Durée de la scolarité + 1 an |
| Contenu de vos feedbacks | 3 ans à compter de la publication |
| Données d'identification (IP chiffrée) | 1 an à compter de la publication |
| Messages et verbatims refusés en modération | 90 jours à compter du refus, puis suppression définitive |
| Évaluations anonymes | Conservées à des fins statistiques |
| Données de présence | Durée de la scolarité + 5 ans (obligation légale) |
| Tokens de session | 3 mois puis suppression automatique |

---

## Modération de vos messages

> ℹ️ **Une relecture humaine avant publication**
>
> Chaque feedback que vous envoyez est d'abord relu par un modérateur. Tant qu'il n'est pas publié, votre message n'est ni diffusé à l'équipe pédagogique ni analysé par le système de classification automatique.
>
> Le texte d'origine sert uniquement à cette relecture : il n'est jamais diffusé et il est effacé au moment de la publication. Le modérateur n'a accès à aucune donnée permettant de vous identifier. Vous pouvez suivre le statut de vos messages (en attente, publié ou refusé avec son motif) directement dans l'application.
>
> Si votre message est refusé, son texte d'origine est immédiatement chiffré (illisible au repos, y compris par nos équipes) et conservé au maximum 90 jours — le temps d'une éventuelle contestation — avant d'être supprimé définitivement. Un message refusé n'ayant jamais été publié, il n'est soumis à aucune obligation de conservation.
>
> Les verbatims que vous laissez dans une évaluation de cours suivent exactement le même parcours.

---

## Conservation légale des contenus (LCEN)

> ℹ️ **Ce que cela signifie pour vos messages**
>
> Conformément à l'article 6-II de la loi n° 2004-575 (LCEN) et au décret n° 2021-1362 du 20 octobre 2021, les données techniques permettant d'identifier l'auteur d'un contenu publié en ligne (adresse IP chiffrée, identifiant technique) sont conservées pendant **1 an** à compter de la publication.
>
> Si vous demandez la suppression d'un feedback, son contenu sera immédiatement effacé de l'affichage. Les données techniques resteront conservées jusqu'à l'expiration du délai légal d'un an, puis seront automatiquement détruites.

---

## Registre de présences et intégrité

> ℹ️ **Pourquoi vos présences ne peuvent pas être supprimées**
>
> Chaque pointage est enregistré dans un registre cryptographique dont les entrées sont chaînées par empreinte SHA-256. Toute modification ou suppression rompt la chaîne et est immédiatement détectable. Ce mécanisme garantit l'authenticité des relevés d'assiduité conformément à l'obligation légale (Art. L123-1 Code de l'éducation).
>
> En application de l'Art. 17.3.b du RGPD, le droit à l'effacement ne s'applique pas à ces données pendant la durée de l'obligation légale. À l'issue de cette période, vos données nominatives sont anonymisées dans notre base ; les empreintes cryptographiques du registre sont conservées sans lien avec votre identité.

---

## Destinataires des données

Vos données sont accessibles uniquement aux personnes habilitées suivantes :

- Équipe pédagogique et administrative de l'IMT Mines Alès (feedbacks modérés)
- Administrateurs de la plateforme (gestion des comptes)
- Système de classification automatique par intelligence artificielle hébergé sur l'infrastructure de l'école (analyse des feedbacks)

Aucune donnée n'est cédée à des tiers commerciaux.

---

## Vos droits

Conformément au RGPD, vous disposez des droits suivants sur vos données personnelles :

**Droit d'accès (Art. 15)** — Obtenir une copie des données vous concernant.

**Droit de rectification (Art. 16)** — Faire corriger des données inexactes.

**Droit à l'effacement (Art. 17)** — Demander la suppression de vos données. Exception : les données de présence et les données techniques (LCEN) sont soumises à des obligations légales de conservation ; elles ne peuvent être effacées pendant la durée applicable (Art. 17.3.b RGPD).

**Droit à la portabilité (Art. 20)** — Recevoir vos données dans un format structuré et lisible.

**Droit d'opposition (Art. 21)** — Vous opposer à certains traitements de vos données.

Pour exercer vos droits, contactez le DPO de l'établissement : dpo@mines-ales.fr

---

## Droit de réclamation

Si vous estimez que le traitement de vos données ne respecte pas la réglementation, vous avez le droit d'introduire une réclamation auprès de la Commission Nationale de l'Informatique et des Libertés (CNIL) — www.cnil.fr

---

*Dernière mise à jour : août 2026 — IMT Mines Alès — Tous droits réservés*

---

*Copie conforme au composant source `apps/mobile/components/apropos-content.tsx` au commit `0dde667` (branche `main`, 2026-08-03).*
