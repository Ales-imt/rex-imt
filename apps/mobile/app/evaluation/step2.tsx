import { ChipGroup } from '@/components/eval/ChipGroup';
import { LikertScale } from '@/components/eval/LikertScale';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const CHIPS_POSITIFS = [
  { id: 'EXEMPLES_CONCRETS', label: 'Exemples concrets' },
  { id: 'THEORIE_SOLIDE',    label: 'Théorie solide' },
  { id: 'ACTUALITE',         label: 'Actualité du domaine' },
  { id: 'CAS_PRATIQUES',     label: 'Cas pratiques' },
];

const CHIPS_NEGATIFS = [
  { id: 'TROP_THEORIQUE', label: 'Trop théorique' },
  { id: 'HORS_PROGRAMME', label: 'Hors programme' },
  { id: 'TROP_DENSE',     label: 'Trop dense' },
  { id: 'PAS_A_JOUR',     label: 'Pas à jour' },
];

export default function EvalStep2Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed = state.scoreContenuAdequation !== null;

  function toggleChip(id: string) {
    const current = state.chipsContenu;
    set({
      chipsContenu: current.includes(id)
        ? current.filter((c) => c !== id)
        : [...current, id],
    });
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Contenu', header: () => <ProgressBar currentStep={3} /> }} />
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Contenu / Programme</Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>
              Adéquation avec le programme annoncé
            </Text>
            <LikertScale
              value={state.scoreContenuAdequation}
              onChange={(v) => set({ scoreContenuAdequation: v })}
              labelMin="Pas du tout"
              labelMax="Parfaitement"
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Points forts</Text>
            <ChipGroup
              chips={CHIPS_POSITIFS}
              selected={state.chipsContenu}
              onToggle={toggleChip}
              polarity="positive"
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Points faibles</Text>
            <ChipGroup
              chips={CHIPS_NEGATIFS}
              selected={state.chipsContenu}
              onToggle={toggleChip}
              polarity="negative"
            />
          </View>
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onPrev={() => router.back()}
            onNext={() => router.push('/evaluation/step3')}
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
    shadowColor: '#000',
    shadowOpacity: 0.04,
    shadowOffset: { width: 0, height: 1 },
    shadowRadius: 3,
    elevation: 1,
  },
  cardLabel: { fontSize: 15, fontWeight: '600' },
});
