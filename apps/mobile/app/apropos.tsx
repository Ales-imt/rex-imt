import { ZenBackground } from '@/components/zen-background';
import { useTheme } from '@/hooks/use-theme';
import { StyleSheet, Text, View } from 'react-native';

export default function AproposScreen() {
  const colors = useTheme();
  return (
    <View style={styles.container}>
      <ZenBackground />
      <Text style={[styles.title, { color: colors.text }]}>À propos</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: 'transparent',
    justifyContent: 'center',
    alignItems: 'center',
  },
  title: {
    fontSize: 24,
    fontWeight: '600',
  },
});
