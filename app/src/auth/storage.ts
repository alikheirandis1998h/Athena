import { Platform } from 'react-native';
import * as SecureStore from 'expo-secure-store';

import type { AuthTokens } from '@/types/user';

const ACCESS_TOKEN_KEY = 'auth_access_token';
const REFRESH_TOKEN_KEY = 'auth_refresh_token';

// expo-secure-store has no web implementation, so the web build falls back to
// localStorage (unencrypted, but there's no browser equivalent of the native secure enclave anyway).
const store = Platform.OS === 'web'
  ? {
      getItemAsync: async (key: string) => {
        try {
          return globalThis.localStorage?.getItem(key) ?? null;
        } catch {
          return null;
        }
      },
      setItemAsync: async (key: string, value: string) => {
        try {
          globalThis.localStorage?.setItem(key, value);
        } catch {
          // ignore (e.g. storage disabled/full)
        }
      },
      deleteItemAsync: async (key: string) => {
        try {
          globalThis.localStorage?.removeItem(key);
        } catch {
          // ignore
        }
      },
    }
  : SecureStore;

export async function getTokens(): Promise<AuthTokens | null> {
  const [accessToken, refreshToken] = await Promise.all([
    store.getItemAsync(ACCESS_TOKEN_KEY),
    store.getItemAsync(REFRESH_TOKEN_KEY),
  ]);

  if (!accessToken || !refreshToken) {
    return null;
  }

  return { accessToken, refreshToken };
}

export async function setTokens(tokens: AuthTokens): Promise<void> {
  await Promise.all([
    store.setItemAsync(ACCESS_TOKEN_KEY, tokens.accessToken),
    store.setItemAsync(REFRESH_TOKEN_KEY, tokens.refreshToken),
  ]);
}

export async function clearTokens(): Promise<void> {
  await Promise.all([
    store.deleteItemAsync(ACCESS_TOKEN_KEY),
    store.deleteItemAsync(REFRESH_TOKEN_KEY),
  ]);
}
