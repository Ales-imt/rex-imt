import { test, expect } from '@playwright/test';

// Programme admin : sélection explicite de la promotion/période seedées, puis
// vérification du calendrier FullCalendar. weekends=false : un samedi ou un
// dimanche, les séances du jour sont hors grille — on saute alors les
// assertions d'affichage (l'assertion API, elle, reste valable).
const WEEKEND = [0, 6].includes(new Date().getDay());

async function ouvrirPlanning(page: import('@playwright/test').Page) {
  await page.goto('/programme/select');
  // Les selects MUI de cette page n'ont pas de nom accessible : on les prend
  // dans l'ordre structurel Année / Promotion / Période.
  const combos = page.getByRole('combobox');
  await combos.nth(1).click();
  await page.getByRole('option', { name: '1A TEST E2E' }).click();
  await combos.nth(2).click();
  await page.getByRole('option', { name: 'S5' }).click();
  await page.getByRole('button', { name: 'Afficher le planning' }).click();
  await page.waitForURL(/\/programme\/\d+/);
}

test('le planning affiche la séance avec sa remarque sous les intervenants', async ({ page }) => {
  test.skip(WEEKEND, 'week-end : la grille masque samedi/dimanche');
  await ouvrirPlanning(page);

  await expect(page.getByText('E2E-B101')).toBeVisible();
  await expect(page.getByText('Réunion DRDV — seed E2E')).toBeVisible();
  await expect(page.getByText('Cours/TD 3 — seed E2E')).toBeVisible();
});

test('la séance sans remarque n\'en affiche pas', async ({ page }) => {
  test.skip(WEEKEND, 'week-end : la grille masque samedi/dimanche');
  await ouvrirPlanning(page);

  // L'événement E2E-A202 est rendu, mais son bloc ne contient que le titre et
  // l'intervenant : une remarque seedée NULL ne doit produire aucune ligne.
  const event = page.locator('.fc-event', { hasText: 'E2E-A202' });
  await expect(event).toBeVisible();
  await expect(event).toContainText('M. BERNARD');
  await expect(event).not.toContainText('seed E2E');
});

test("l'API /planning/reservation porte remarque et groupes", async ({ page }) => {
  const repPromise = page.waitForResponse(r => r.url().includes('/planning/reservation') && r.ok());
  await ouvrirPlanning(page);
  const reservations: Array<{
    salles: Array<{ name: string }>;
    groupes: Array<{ name: string }>;
    remarque: string | null;
  }> = await (await repPromise).json();

  const avecGroupe = reservations.find(r => r.salles.some(s => s.name === 'E2E-B101'));
  expect(avecGroupe?.remarque).toBe('Réunion DRDV — seed E2E');
  expect(avecGroupe?.groupes.map(g => g.name)).toEqual(['G1-E2E']);

  const promoEntiere = reservations.find(r => r.salles.some(s => s.name === 'E2E-A202'));
  expect(promoEntiere?.remarque).toBeNull();
  expect(promoEntiere?.groupes).toEqual([]);
});
