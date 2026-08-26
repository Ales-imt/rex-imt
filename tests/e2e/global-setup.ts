import { execSync } from 'node:child_process';
import * as path from 'node:path';

const PG_CONTAINER = 'postgres-16.10-alpine-rex';
const LDAP_CONTAINER = 'openldap-rex';

// Comptes dont le mot de passe est réaligné sur « password » avant chaque run.
// Le volume OpenLDAP survit aux `make local` : un mot de passe changé à la main
// y persiste et ferait échouer les logins de test en silence.
const LDAP_UIDS = ['clement.trens', 'bob.dupont', 'claire.lecocq', 'jean.bernard'];

export default function globalSetup() {
  // Échoue tôt et clairement si la stack n'est pas debout.
  try {
    execSync(`docker exec ${PG_CONTAINER} pg_isready -U postgres`, { stdio: 'pipe' });
  } catch {
    throw new Error(
      `La stack locale ne répond pas (conteneur ${PG_CONTAINER}). Lancer \`make local\` (ou \`make release-local\`) d'abord.`
    );
  }

  const seed = path.join(__dirname, 'seed.sql');
  execSync(`docker exec -i ${PG_CONTAINER} psql -U postgres -d db_rex -v ON_ERROR_STOP=1 < ${JSON.stringify(seed)}`, {
    shell: '/bin/sh',
    stdio: 'pipe',
  });

  for (const uid of LDAP_UIDS) {
    execSync(
      `docker exec ${LDAP_CONTAINER} ldappasswd -x -D "cn=admin,dc=ema,dc=fr" -w adminpassword -s password "uid=${uid},ou=people,dc=ema,dc=fr"`,
      { stdio: 'pipe' }
    );
  }
}
