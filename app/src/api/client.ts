import Constants from 'expo-constants';
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';

import { clearTokens, getTokens, setTokens } from '@/auth/storage';
import type { RefreshResponse } from '@/types/user';

const apiUrl =
  (Constants.expoConfig?.extra?.apiUrl as string | undefined) ?? 'http://localhost:8080';

export const client = axios.create({
  baseURL: `${apiUrl}/api/v1`,
});

let onLogout: (() => void) | null = null;

export function setLogoutHandler(handler: () => void): void {
  onLogout = handler;
}

client.interceptors.request.use(async (config) => {
  const tokens = await getTokens();
  if (tokens?.accessToken) {
    config.headers.set('Authorization', `Bearer ${tokens.accessToken}`);
  }
  return config;
});

interface RetriableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean;
}

// Guards against multiple simultaneous 401s each firing their own refresh call:
// the first 401 starts the refresh and stores its promise; concurrent 401s await
// the same promise instead of hitting /auth/refresh again.
let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(): Promise<string> {
  const tokens = await getTokens();
  if (!tokens?.refreshToken) {
    throw new Error('No refresh token available');
  }

  const response = await axios.post<RefreshResponse>(`${apiUrl}/api/v1/auth/refresh`, {
    refresh_token: tokens.refreshToken,
  });

  const newTokens = {
    accessToken: response.data.access_token,
    refreshToken: response.data.refresh_token,
  };
  await setTokens(newTokens);
  return newTokens.accessToken;
}

client.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as RetriableConfig | undefined;

    if (error.response?.status !== 401 || !originalRequest || originalRequest._retried) {
      return Promise.reject(error);
    }

    originalRequest._retried = true;

    try {
      refreshPromise ??= refreshAccessToken().finally(() => {
        refreshPromise = null;
      });
      const accessToken = await refreshPromise;
      originalRequest.headers.set('Authorization', `Bearer ${accessToken}`);
      return client(originalRequest);
    } catch (refreshError) {
      await clearTokens();
      onLogout?.();
      return Promise.reject(refreshError);
    }
  },
);
