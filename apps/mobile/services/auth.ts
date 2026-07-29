export const API_BASE = process.env.EXPO_PUBLIC_API_BASE ?? 'http://10.24.182.46:3300/api/v2';

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

export async function requestEmailCode(email: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/email/request`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.message ?? `Erreur ${res.status}`);
  }
}

export async function verifyEmailCode(email: string, code: string): Promise<LoginResponse> {
  const res = await fetch(`${API_BASE}/auth/email/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, code }),
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
  const { File, Paths } = await import('expo-file-system');
  try {
    await apiInstance.get('/auth/logout');
  } finally {
    await clearTokens();
    // Cast nécessaire : le .d.ts de expo-file-system@19 ne reporte pas les membres
    // hérités (exists/delete) de File, bien qu'ils existent au runtime.
    const chatFile = new File(Paths.document, 'chat_messages.json') as InstanceType<typeof File> & {
      exists: boolean;
      delete(): void;
    };
    if (chatFile.exists) chatFile.delete();
  }
}
