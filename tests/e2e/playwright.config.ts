import { defineConfig } from '@playwright/test';

// Cibles : les conteneurs de la stack locale (`make local`), qui servent les
// fronts TELS QUE DÉPLOYÉS (nginx + build de prod), pas les serveurs de dev.
export const ELEVE_URL = process.env.E2E_ELEVE_URL ?? 'http://10.20.1.11:8131';
export const ADMIN_URL = process.env.E2E_ADMIN_URL ?? 'http://10.20.1.10:8121';

// Comptes de l'annuaire LDAP de test (infras/container/ldap/bootstrap).
// global-setup réaligne leurs mots de passe à chaque run : le volume OpenLDAP
// persiste entre les `make local` et peut dériver du LDIF de bootstrap.
export const ELEVE_EMAIL = 'clement.trens@etu.mines-ales.fr';
export const ADMIN_EMAIL = 'claire.lecocq@mines-ales.fr';
export const PASSWORD = 'password';

export default defineConfig({
  testDir: '.',
  globalSetup: './global-setup.ts',
  // Un seul worker : les deux fronts partagent la même base seedée, et les
  // tests élève dépendent de l'état localStorage écrit par leur setup.
  workers: 1,
  timeout: 30_000,
  expect: { timeout: 10_000 },
  reporter: [['list'], ['html', { open: 'never' }]],
  projects: [
    {
      name: 'setup-eleve',
      testMatch: 'eleve.setup.ts',
      use: { baseURL: ELEVE_URL },
    },
    {
      name: 'eleve',
      testMatch: 'eleve.spec.ts',
      dependencies: ['setup-eleve'],
      use: { baseURL: ELEVE_URL, storageState: '.auth/eleve.json' },
    },
    {
      name: 'setup-admin',
      testMatch: 'admin.setup.ts',
      use: { baseURL: ADMIN_URL },
    },
    {
      name: 'admin',
      testMatch: 'admin.spec.ts',
      dependencies: ['setup-admin'],
      use: { baseURL: ADMIN_URL, storageState: '.auth/admin.json' },
    },
  ],
});
