import { StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';

export default function LicenseScreen() {
  return (
    <ThemedView style={styles.container}>
      <ThemedText type="title">Hello 👋</ThemedText>
      <ThemedText type="subtitle">Bienvenue !</ThemedText>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    gap: 12,
  },
});
