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
