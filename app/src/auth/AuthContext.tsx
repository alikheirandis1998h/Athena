import { createContext, useCallback, useContext, useEffect, useMemo, useReducer } from 'react';
import type { PropsWithChildren } from 'react';

import * as authApi from '@/api/auth';
import { setLogoutHandler } from '@/api/client';
import { clearTokens, getTokens, setTokens } from '@/auth/storage';
import type { User } from '@/types/user';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isLoading: boolean;
}

type AuthAction =
  | { type: 'RESTORE'; user: User | null; accessToken: string | null }
  | { type: 'SIGNED_IN'; user: User; accessToken: string }
  | { type: 'SIGNED_OUT' };

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'RESTORE':
      return { user: action.user, accessToken: action.accessToken, isLoading: false };
    case 'SIGNED_IN':
      return { user: action.user, accessToken: action.accessToken, isLoading: false };
    case 'SIGNED_OUT':
      return { user: null, accessToken: null, isLoading: false };
  }
}

interface AuthContextValue extends AuthState {
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [state, dispatch] = useReducer(authReducer, {
    user: null,
    accessToken: null,
    isLoading: true,
  });

  const signOut = useCallback(async () => {
    await clearTokens();
    dispatch({ type: 'SIGNED_OUT' });
  }, []);

  useEffect(() => {
    setLogoutHandler(() => {
      dispatch({ type: 'SIGNED_OUT' });
    });
  }, []);

  useEffect(() => {
    (async () => {
      const tokens = await getTokens();
      if (!tokens) {
        dispatch({ type: 'RESTORE', user: null, accessToken: null });
        return;
      }

      try {
        const user = await authApi.getMe();
        dispatch({ type: 'RESTORE', user, accessToken: tokens.accessToken });
      } catch {
        await clearTokens();
        dispatch({ type: 'RESTORE', user: null, accessToken: null });
      }
    })();
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const response = await authApi.login(email, password);
    await setTokens({
      accessToken: response.access_token,
      refreshToken: response.refresh_token,
    });
    dispatch({ type: 'SIGNED_IN', user: response.user, accessToken: response.access_token });
  }, []);

  const register = useCallback(async (email: string, password: string) => {
    const response = await authApi.register(email, password);
    await setTokens({
      accessToken: response.access_token,
      refreshToken: response.refresh_token,
    });
    dispatch({ type: 'SIGNED_IN', user: response.user, accessToken: response.access_token });
  }, []);

  const logout = useCallback(async () => {
    const tokens = await getTokens();
    if (tokens?.refreshToken) {
      try {
        await authApi.logout(tokens.refreshToken);
      } catch {
        // Best-effort: proceed with local sign-out even if the server call fails.
      }
    }
    await signOut();
  }, [signOut]);

  const value = useMemo<AuthContextValue>(
    () => ({
      ...state,
      isAuthenticated: state.user !== null,
      login,
      register,
      logout,
    }),
    [state, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
