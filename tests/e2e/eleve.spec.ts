import { test, expect } from '@playwright/test';

// Programme élève : les séances seedées (tests/e2e/seed.sql) sont ancrées sur
// le JOUR courant — l'écran s'ouvre dessus, aucun besoin de naviguer.

test.beforeEach(async ({ page }) => {
  await page.goto('/programme');
});

test('la séance de groupe affiche salle · prof · groupe et la remarque', async ({ page }) => {
  // La ligne méta complète, en un seul texte : prouve l'ordre et le séparateur.
  await expect(page.getByText('E2E-B101 · M. BERNARD · G1-E2E')).toBeVisible();
  await expect(page.getByText('Réunion DRDV — seed E2E')).toBeVisible();
});

test("la séance de promotion entière n'affiche ni groupe ni remarque", async ({ page }) => {
  // Le texte EXACT sans suffixe « · G1-E2E » prouve a contrario que rien ne
  // s'ajoute quand groupe_id est NULL.
  await expect(page.getByText('E2E-A202 · M. BERNARD', { exact: true })).toBeVisible();
});

test("l'API /programme porte groupe et remarque", async ({ page }) => {
  const rep = await page.waitForResponse(r => r.url().includes('/api/v2/programme') && r.ok());
  const cours: Array<{ salle: string; groupe: string; remarque: string }> = await rep.json();

  const avecGroupe = cours.find(c => c.salle === 'E2E-B101');
  expect(avecGroupe).toMatchObject({ groupe: 'G1-E2E', remarque: 'Réunion DRDV — seed E2E' });

  const promoEntiere = cours.find(c => c.salle === 'E2E-A202');
  expect(promoEntiere).toMatchObject({ groupe: '', remarque: '' });
});

test('le lendemain montre sa séance avec groupe et remarque', async ({ page }) => {
  const demain = new Date();
  demain.setDate(demain.getDate() + 1);
  // La barre de semaine n'affiche que la semaine courante : un dimanche, le
  // lendemain est hors bande, le test n'a pas de cellule à cliquer.
  test.skip(new Date().getDay() === 0, 'dimanche : le lendemain est sur la semaine suivante');

  await page.getByText(String(demain.getDate()), { exact: true }).first().click();
  await expect(page.getByText('E2E-C303 · M. BERNARD · G1-E2E')).toBeVisible();
  await expect(page.getByText('Cours/TD 3 — seed E2E')).toBeVisible();
});

// Multi-onglets : deux pages du même contexte partagent localStorage ET les
// Web Locks. On force le rafraîchissement en réécrivant l'exp de l'access
// token sous le seuil des 30 s (le refresh token, lui, reste intact — c'est
// lui que le serveur vérifie), puis on navigue dans les deux onglets en même
// temps. Attendu : UN seul POST /auth/refresh au total, et les deux onglets
// gardent leur session — sans le verrou inter-onglets, le perdant du refresh
// concurrent recevait un 400 et effaçait la session fraîche du gagnant.
//
// EN DERNIER dans ce fichier : le refresh consomme (rotation stricte) le jeton
// sauvé par le setup ; un test placé après repartirait du storageState avec un
// refresh token déjà tourné et se ferait déconnecter au premier renouvellement.
test('deux onglets simultanés ne déclenchent qu\'un seul refresh', async ({ context, page }) => {
  await page.goto('/programme');
  await expect(page.getByText('E2E-B101 · M. BERNARD · G1-E2E')).toBeVisible();

  await page.evaluate(() => {
    const jeton = localStorage.getItem('access_token');
    if (!jeton) throw new Error('access_token absent');
    const [h, p, s] = jeton.split('.');
    const depadde = p.replace(/-/g, '+').replace(/_/g, '/');
    const charge = JSON.parse(atob(depadde));
    charge.exp = Math.floor(Date.now() / 1000) + 5; // sous le seuil de 30 s
    const reencode = btoa(JSON.stringify(charge)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    localStorage.setItem('access_token', `${h}.${reencode}.${s}`);
  });

  const page2 = await context.newPage();
  let refreshs = 0;
  for (const p of [page, page2]) {
    p.on('request', r => { if (r.url().includes('/auth/refresh')) refreshs++; });
  }

  await Promise.all([page.goto('/programme'), page2.goto('/programme')]);

  await expect(page.getByText('E2E-B101 · M. BERNARD · G1-E2E')).toBeVisible();
  await expect(page2.getByText('E2E-B101 · M. BERNARD · G1-E2E')).toBeVisible();
  expect(refreshs).toBe(1);
  await page2.close();
});
