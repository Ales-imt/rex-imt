# Tests E2E — stack locale

Vérifient les deux fronts **tels que déployés** (nginx des conteneurs locaux) :
le programme élève (`http://10.20.1.11:8131`) et le planning admin
(`http://10.20.1.10:8121`), avec l'affichage du groupe et de la remarque des
séances côté écran ET côté API.

## Prérequis

- La stack locale debout : `make local` (première fois) ou `make release-local`.
- Docker accessible sans sudo (le seed passe par `docker exec`).

## Lancer

```sh
cd tests/e2e
npm install          # première fois : installe aussi le navigateur
npx playwright install chromium
npm test
```

`npm run report` ouvre le rapport HTML du dernier run.

## Ce que fait chaque run

1. `global-setup.ts` : applique `seed.sql` (comptes + promotion/période/matière/
   groupe + 3 séances ancrées sur le JOUR courant, salles `E2E-*`) et réaligne
   les mots de passe LDAP des comptes de test sur `password` — le volume
   OpenLDAP persiste entre les `make local` et peut dériver du LDIF.
2. Deux projets de setup se connectent par l'UI (élève : `clement.trens`,
   admin : `claire.lecocq`) et sauvent l'état de session dans `.auth/`.
3. Les specs `eleve.spec.ts` / `admin.spec.ts` vérifient l'affichage et les
   réponses API.

## Limites connues

- Le calendrier admin masque les week-ends et l'écran élève ouvre sur
  aujourd'hui : les assertions d'affichage se sautent d'elles-mêmes le
  week-end (celles sur l'API restent actives).
- Le scanner caméra (Pointage) et les comportements natifs Android ne sont pas
  couverts — hors de portée d'un navigateur.
