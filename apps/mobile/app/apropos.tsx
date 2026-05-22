import { AproposContent } from '@/components/apropos-content';
import { ZenBackground } from '@/components/zen-background';
import { useTheme } from '@/hooks/use-theme';
import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function AproposScreen() {
  const colors = useTheme();

  return (
    <SafeAreaView style={[styles.page, { backgroundColor: colors.pageBg }]} edges={['bottom']}>
      <ZenBackground />
      <AproposContent colors={colors} />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1 },
});
