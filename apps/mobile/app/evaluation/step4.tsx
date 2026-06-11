import { ChipGroup } from '@/components/eval/ChipGroup';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { StarRating } from '@/components/eval/StarRating';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const CHIPS_SUPPORTS = [
  { id: 'PEU_EXEMPLES',      label: "Peu d'exemples" },
  { id: 'SLIDES_ILLISIBLES', label: 'Slides illisibles' },
  { id: 'PAS_A_JOUR',        label: 'Pas à jour' },
  { id: 'INCOMPLET',         label: 'Incomplet' },
  { id: 'PAS_DE_SUPPORT',    label: 'Pas de support' },
];

export default function EvalStep4Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed = state.scoreSupports !== null;

  function toggleChip(id: string) {
    const current = state.chipsSupports;
    set({
      chipsSupports: current.includes(id)
        ? current.filter((c) => c !== id)
        : [...current, id],
    });
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Supports', header: () => <ProgressBar currentStep={5} /> }} />
      <KeyboardAvoidingView
        style={[styles.container, { backgroundColor: colors.pageBg }]}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Supports de cours</Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Note globale</Text>
            <StarRating
              value={state.scoreSupports}
              onChange={(v) => set({ scoreSupports: v })}
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Points à améliorer</Text>
            <ChipGroup
              chips={CHIPS_SUPPORTS}
              selected={state.chipsSupports}
              onToggle={toggleChip}
              polarity="negative"
            />
          </View>

          <TextInput
            style={[styles.textInput, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder, color: colors.textPrimary }]}
            placeholder="Un commentaire sur les supports ? (optionnel)"
            placeholderTextColor={colors.inputPlaceholder}
            value={state.verbatimSupports}
            onChangeText={(t) => set({ verbatimSupports: t })}
            multiline
            maxLength={300}
            textAlignVertical="top"
          />
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onPrev={() => router.back()}
            onNext={() => router.push('/evaluation/step5')}
            nextDisabled={!canProceed}
          />
        </View>
      </KeyboardAvoidingView>
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
  textInput: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    fontSize: 14,
    minHeight: 90,
  },
});
