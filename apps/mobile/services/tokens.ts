import * as SecureStore from 'expo-secure-store';

const ACCESS_KEY = 'access_token';
const REFRESH_KEY = 'refresh_token';
const PSEUDO_KEY = 'pseudo';

export function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

export async function getOrCreatePseudo(): Promise<string> {
  let pseudo = await SecureStore.getItemAsync(PSEUDO_KEY);
  if (!pseudo) {
    pseudo = generateUUID();
    await SecureStore.setItemAsync(PSEUDO_KEY, pseudo);
  }
  return pseudo;
}

export async function getPseudo(): Promise<string | null> {
  return SecureStore.getItemAsync(PSEUDO_KEY);
}

export async function saveTokens(access: string, refresh: string): Promise<void> {
  await SecureStore.setItemAsync(ACCESS_KEY, access);
  await SecureStore.setItemAsync(REFRESH_KEY, refresh);
}

export async function getAccessToken(): Promise<string | null> {
  return SecureStore.getItemAsync(ACCESS_KEY);
}

export async function getRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(REFRESH_KEY);
}

export async function clearTokens(): Promise<void> {
  await SecureStore.deleteItemAsync(ACCESS_KEY);
  await SecureStore.deleteItemAsync(REFRESH_KEY);
  await SecureStore.deleteItemAsync(PSEUDO_KEY);
}

export function isExpiringSoon(token: string, thresholdSec = 30): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return Date.now() / 1000 > (payload.exp ?? 0) - thresholdSec;
  } catch {
    return true;
  }
}
