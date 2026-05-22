import { AproposContent } from '@/components/apropos-content';
import { ZenBackground } from '@/components/zen-background';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { StyleSheet, Text, TouchableOpacity } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function AproposFirstLoginScreen() {
  const colors = useTheme();
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  async function handleContinue() {
    setLoading(true);
    try {
      await apiInstance.patch('/me/informed');
    } finally {
      setLoading(false);
      router.replace('/agora');
    }
  }

  return (
    <SafeAreaView style={[styles.page, { backgroundColor: colors.pageBg }]} edges={['bottom']}>
      <ZenBackground />
      <AproposContent colors={colors}>
        <TouchableOpacity
          style={[styles.continueBtn, { backgroundColor: colors.tint }, loading && { opacity: 0.6 }]}
          onPress={handleContinue}
          activeOpacity={0.85}
          disabled={loading}
        >
          <Text style={[styles.continueBtnText, { color: colors.background }]}>
            {loading ? 'Chargement…' : 'J\'ai compris, continuer'}
          </Text>
        </TouchableOpacity>
      </AproposContent>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1 },
  continueBtn: {
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 24,
  },
  continueBtnText: {
    fontSize: 16,
    fontWeight: '600',
    letterSpacing: 0.3,
  },
});
