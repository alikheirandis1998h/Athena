import { StyleSheet, Text, View } from 'react-native';

import { useAuth } from '@/auth/AuthContext';

export default function HomeScreen() {
  const { user } = useAuth();

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Welcome{user ? `, ${user.email}` : ''}</Text>
      <Text style={styles.subtitle}>You are signed in.</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    paddingHorizontal: 24,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
  },
});
