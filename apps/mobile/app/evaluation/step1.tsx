import { LikertScale } from '@/components/eval/LikertScale';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { StarRating } from '@/components/eval/StarRating';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

export default function EvalStep1Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed = state.scorePedaGlobal !== null && state.scorePedaClarte !== null;

  return (
    <>
      <Stack.Screen options={{ title: 'Pédagogie', header: () => <ProgressBar currentStep={2} /> }} />
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Qualité pédagogique</Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Note globale</Text>
            <StarRating
              value={state.scorePedaGlobal}
              onChange={(v) => set({ scorePedaGlobal: v })}
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Clarté des explications</Text>
            <LikertScale
              value={state.scorePedaClarte}
              onChange={(v) => set({ scorePedaClarte: v })}
              labelMin="Pas du tout"
              labelMax="Tout à fait"
            />
          </View>
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onPrev={() => router.back()}
            onNext={() => router.push('/evaluation/step2')}
            nextDisabled={!canProceed}
          />
        </View>
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, gap: 16 },
  title: { fontSize: 20, fontWeight: '700', marginBottom: 4 },
  card: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 16,
    gap: 14,
    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    elevation: 1,
  },
  cardLabel: { fontSize: 15, fontWeight: '600' },
});
