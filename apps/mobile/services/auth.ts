export const API_BASE = 'http://10.13.6.46:3300/api/v2';

export type LoginResponse = {
  name: string;
  surname: string;
  roles: string[];
  access_token: string;
  refresh_token: string;
};

export async function login(identifiant: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifiant, password }),
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.message ?? `Erreur ${res.status}`);
  }

  return res.json();
}

export async function logout(): Promise<void> {
  // Import dynamique pour éviter la dépendance circulaire auth ↔ api
  const { apiInstance } = await import('./api');
  const { clearTokens } = await import('./tokens');
  try {
    await apiInstance.get('/auth/logout');
  } finally {
    await clearTokens();
  }
}
