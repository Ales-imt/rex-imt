import { test as setup, expect } from '@playwright/test';
import { ELEVE_EMAIL, PASSWORD } from './playwright.config';

// Login élève via l'UI, état de session sauvé pour les specs.
//
// Le front élève est du React Native Web : les « boutons » sont des div
// cliquables sans rôle ARIA, on les vise par leur texte exact.
//
// Au premier login d'un compte, deux écrans s'intercalent avant /programme :
// /pseudo-setup (pseudo anonyme) puis /apropos-first-login (RGPD). Ils ne
// réapparaissent pas ensuite — le parcours les franchit donc SI présents.
setup('login élève', async ({ page }) => {
  await page.goto('/');

  await page.getByRole('textbox').fill(ELEVE_EMAIL);
  await page.getByText('Continuer', { exact: true }).click();

  await page.getByRole('textbox').fill(PASSWORD);
  await page.getByText('Se connecter', { exact: true }).click();

  await page.waitForURL(/\/(programme|pseudo-setup|apropos-first-login)/);

  if (page.url().includes('pseudo-setup')) {
    await page.getByText("J'ai noté mon pseudo, continuer").click();
  }
  if (page.url().includes('apropos-first-login')) {
    await page.getByText("J'ai compris, continuer").click();
  }

  await page.waitForURL(/\/programme/);
  await expect(page.getByText('Programme', { exact: true })).toBeVisible();

  await page.context().storageState({ path: '.auth/eleve.json' });
});
