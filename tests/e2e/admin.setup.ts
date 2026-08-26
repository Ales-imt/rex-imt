import { test as setup, expect } from '@playwright/test';
import { ADMIN_EMAIL, PASSWORD } from './playwright.config';

// Login gestionnaire sur web-admin (formulaire MUI classique), état sauvé.
setup('login admin', async ({ page }) => {
  await page.goto('/login');

  await page.getByRole('textbox', { name: 'Adresse e-mail' }).fill(ADMIN_EMAIL);
  await page.getByRole('textbox', { name: /mot de passe/i }).fill(PASSWORD);
  await page.getByRole('button', { name: 'Se connecter' }).click();

  // La barre de navigation filtrée par rôles prouve le login ET le rôle.
  await expect(page.getByRole('link', { name: 'Programme' })).toBeVisible();

  await page.context().storageState({ path: '.auth/admin.json' });
});
