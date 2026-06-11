import { ChipGroup } from '@/components/eval/ChipGroup';
import { LikertScale } from '@/components/eval/LikertScale';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const CHIPS_POSITIFS = [
  { id: 'DISPONIBLE',   label: 'Disponible' },
  { id: 'A_ECOUTE',     label: 'À l\'écoute' },
  { id: 'DYNAMIQUE',    label: 'Dynamique' },
  { id: 'BIENVEILLANT', label: 'Bienveillant' },
];

const CHIPS_NEGATIFS = [
  { id: 'PEU_DISPONIBLE', label: 'Peu disponible' },
  { id: 'RYTHME_IMPOSE',  label: 'Rythme imposé' },
  { id: 'PEU_ECOUTE',     label: "Peu à l'écoute" },
];

export default function EvalStep5Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed = state.scoreAmbiance !== null;

  function toggleChip(id: string) {
    const current = state.chipsAmbiance;
    set({
      chipsAmbiance: current.includes(id)
        ? current.filter((c) => c !== id)
        : [...current, id],
    });
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Ambiance', header: () => <ProgressBar currentStep={6} /> }} />
      <KeyboardAvoidingView
        style={[styles.container, { backgroundColor: colors.pageBg }]}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>
            Ambiance & relation prof–étudiants
          </Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Ambiance générale</Text>
            <LikertScale
              value={state.scoreAmbiance}
              onChange={(v) => set({ scoreAmbiance: v })}
              labelMin="Très mauvaise"
              labelMax="Excellente"
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Points forts</Text>
            <ChipGroup
              chips={CHIPS_POSITIFS}
              selected={state.chipsAmbiance}
              onToggle={toggleChip}
              polarity="positive"
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Points faibles</Text>
            <ChipGroup
              chips={CHIPS_NEGATIFS}
              selected={state.chipsAmbiance}
              onToggle={toggleChip}
              polarity="negative"
            />
          </View>

          <TextInput
            style={[styles.textInput, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder, color: colors.textPrimary }]}
            placeholder="Un commentaire ? (optionnel)"
            placeholderTextColor={colors.inputPlaceholder}
            value={state.verbatimAmbiance}
            onChangeText={(t) => set({ verbatimAmbiance: t })}
            multiline
            maxLength={300}
            textAlignVertical="top"
          />
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onPrev={() => router.back()}
            onNext={() => router.push('/evaluation/step6')}
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
