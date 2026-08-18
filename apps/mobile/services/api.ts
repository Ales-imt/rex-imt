import axios from 'axios';
import { router } from 'expo-router';
import { API_BASE } from './auth';
import { clearTokens, getAccessToken, getRefreshToken, isExpiringSoon, saveTokens } from './tokens';

export const apiInstance = axios.create({
  baseURL: API_BASE,
});

let refreshPromise: Promise<void> | null = null;

async function doRefresh(): Promise<void> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) throw new Error('No refresh token');
  const res = await axios.post(`${API_BASE}/auth/refresh`, { refreshToken });
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
        if (!refreshPromise) {
          refreshPromise = doRefresh().finally(() => { refreshPromise = null; });
        }
        await refreshPromise;
      }
      const current = await getAccessToken();
      if (current) {
        config.headers.Authorization = `Bearer ${current}`;
      }
    } catch {
      await clearTokens();
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
