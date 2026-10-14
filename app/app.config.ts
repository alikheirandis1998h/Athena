import type { ConfigContext, ExpoConfig } from 'expo/config';

import baseConfig from './app.json';

const DEFAULT_API_URL = 'http://localhost:8080';

export default ({ config }: ConfigContext): ExpoConfig => ({
  ...(baseConfig.expo as ExpoConfig),
  ...config,
  extra: {
    ...config.extra,
    apiUrl: process.env.EXPO_PUBLIC_API_URL ?? DEFAULT_API_URL,
  },
});
