import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const ACCESS_KEY = 'access_token';
const REFRESH_KEY = 'refresh_token';
const PSEUDO_KEY = 'pseudo';

// expo-secure-store ne fonctionne pas sur web : fallback localStorage
const storage = {
  async getItem(key: string): Promise<string | null> {
    if (Platform.OS === 'web') return localStorage.getItem(key);
    return SecureStore.getItemAsync(key);
  },
  async setItem(key: string, value: string): Promise<void> {
    if (Platform.OS === 'web') { localStorage.setItem(key, value); return; }
    return SecureStore.setItemAsync(key, value);
  },
  async deleteItem(key: string): Promise<void> {
    if (Platform.OS === 'web') { localStorage.removeItem(key); return; }
    return SecureStore.deleteItemAsync(key);
  },
};

export function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

export async function getOrCreatePseudo(): Promise<string> {
  let pseudo = await storage.getItem(PSEUDO_KEY);
  if (!pseudo) {
    pseudo = generateUUID();
    await storage.setItem(PSEUDO_KEY, pseudo);
  }
  return pseudo;
}

export async function getPseudo(): Promise<string | null> {
  return storage.getItem(PSEUDO_KEY);
}

export async function saveTokens(access: string, refresh: string): Promise<void> {
  await storage.setItem(ACCESS_KEY, access);
  await storage.setItem(REFRESH_KEY, refresh);
}

export async function getAccessToken(): Promise<string | null> {
  return storage.getItem(ACCESS_KEY);
}

export async function getRefreshToken(): Promise<string | null> {
  return storage.getItem(REFRESH_KEY);
}

export async function clearTokens(): Promise<void> {
  await storage.deleteItem(ACCESS_KEY);
  await storage.deleteItem(REFRESH_KEY);
  await storage.deleteItem(PSEUDO_KEY);
}

export function isExpiringSoon(token: string, thresholdSec = 30): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return Date.now() / 1000 > (payload.exp ?? 0) - thresholdSec;
  } catch {
    return true;
  }
}
