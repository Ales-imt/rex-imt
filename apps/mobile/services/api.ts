import axios from 'axios';
import { router } from 'expo-router';
import { API_BASE } from './auth';
import { clearTokens, getAccessToken, getRefreshToken, isExpiringSoon, saveTokens } from './tokens';

export const apiInstance = axios.create({
  baseURL: API_BASE,
});

// Mutex partagé avec services/session.ts (rafraichir()) : au redémarrage de
// l'app, la garde de navigation (verifierSession) et la première requête d'un
// écran peuvent toutes deux voir l'access token expirant au même instant.
// Le refresh token est à usage unique côté serveur (rotation stricte, sans
// grâce) : deux appels concurrents à /auth/refresh font perdre l'un des deux,
// qui recevrait un refus et effacerait une session que l'autre vient
// pourtant de renouveler avec succès.
let refreshPromise: Promise<void> | null = null;

export async function ensureRefreshed(): Promise<void> {
  if (!refreshPromise) {
    refreshPromise = doRefresh().finally(() => { refreshPromise = null; });
  }
  return refreshPromise;
}

// Sur web, le mutex ci-dessus ne protège qu'UN onglet : localStorage est
// partagé mais chaque onglet a son propre runtime. Deux onglets rafraîchissant
// ensemble consommeraient le même refresh token (usage unique côté serveur) :
// le perdant recevrait un 400 et son intercepteur effacerait la session que le
// gagnant vient pourtant d'écrire — déconnexion des deux onglets. Web Locks
// sérialise les onglets sur un même verrou nommé ; une fois le verrou obtenu,
// on REVÉRIFIE le jeton : si un autre onglet a rafraîchi pendant l'attente, le
// jeton relu de localStorage est frais et il n'y a plus rien à faire.
// Sur natif, navigator.locks n'existe pas — un seul runtime, le mutex suffit.
async function doRefresh(): Promise<void> {
  const locks: LockManager | undefined = (globalThis.navigator as Navigator | undefined)?.locks;
  if (!locks) return appelRefresh();
  return locks.request('rex-auth-refresh', async () => {
    const token = await getAccessToken();
    if (token && !isExpiringSoon(token)) return;
    await appelRefresh();
  });
}

async function appelRefresh(): Promise<void> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) throw new Error('No refresh token');
  const res = await axios.post(`${API_BASE}/auth/refresh`, { refreshToken }, { timeout: 8_000 });
  await saveTokens(res.data.accessToken, res.data.refreshToken);
}

// Intercepteurs posés à l'import du module, et non depuis un effet d'écran :
// React vide les effets des enfants avant ceux du parent, si bien qu'un écran
// monté à froid lançait sa requête avant que le RootLayout n'ait installé quoi
// que ce soit — donc sans Authorization, donc 401 sur une session valide.
// L'import précède forcément le premier appel, l'ordre de montage n'entre plus
// en jeu. Poser un intercepteur n'a pas d'autre effet de bord.
function setupAxiosInterceptors() {
  apiInstance.interceptors.request.use(async (config) => {
    try {
      const token = await getAccessToken();
      if (token && isExpiringSoon(token)) {
        await ensureRefreshed();
      }
      const current = await getAccessToken();
      if (current) {
        config.headers.Authorization = `Bearer ${current}`;
      }
    } catch (e) {
      // Seul un refus du serveur (4xx : jeton de renouvellement inconnu,
      // révoqué, compte banni) clôt la session. Une panne réseau ne dit rien
      // de sa validité : effacer les jetons renverrait au login un utilisateur
      // valide passé hors ligne. La requête part alors sans en-tête, échoue,
      // et sera rejouée plus tard — c'est réversible, la déconnexion non.
      const statut = axios.isAxiosError(e) ? e.response?.status : undefined;
      const panneReseau = axios.isAxiosError(e) && !e.response;
      if (!panneReseau && (statut === undefined || statut < 500)) {
        await clearTokens();
      }
    }
    return config;
  });

  apiInstance.interceptors.response.use(
    res => res,
    async (error) => {
      if (error.response?.status === 401) {
        await clearTokens();
        router.replace('/');
      }
      return Promise.reject(error);
    }
  );

}

setupAxiosInterceptors();
