import { client } from '@/api/client';
import type { AuthResponse, RefreshResponse, User } from '@/types/user';

export async function register(email: string, password: string): Promise<AuthResponse> {
  const response = await client.post<AuthResponse>('/auth/register', { email, password });
  return response.data;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const response = await client.post<AuthResponse>('/auth/login', { email, password });
  return response.data;
}

export async function refresh(refreshToken: string): Promise<RefreshResponse> {
  const response = await client.post<RefreshResponse>('/auth/refresh', {
    refresh_token: refreshToken,
  });
  return response.data;
}

export async function logout(refreshToken: string): Promise<void> {
  await client.post('/auth/logout', { refresh_token: refreshToken });
}

export async function getMe(): Promise<User> {
  const response = await client.get<{ user: User }>('/users/me');
  return response.data.user;
}
