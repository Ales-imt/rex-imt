import axios from 'axios';
import { API_BASE } from './auth';
import { clearTokens, getAccessToken, getRefreshToken, isExpiringSoon, saveTokens } from './tokens';

export const apiInstance = axios.create({
  baseURL: API_BASE,
});

let axiosInit = false;
let refreshPromise: Promise<void> | null = null;

async function doRefresh(): Promise<void> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) throw new Error('No refresh token');
  const res = await axios.post(`${API_BASE}/auth/refresh`, { refreshToken });
  await saveTokens(res.data.accessToken, res.data.refreshToken);
}

export function setupAxiosInterceptors() {
  if (axiosInit) return;

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

  axiosInit = true;
}
