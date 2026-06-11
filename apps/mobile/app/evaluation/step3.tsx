import { ChipGroup } from '@/components/eval/ChipGroup';
import { LikertScale } from '@/components/eval/LikertScale';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import React from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const CHARGE_OPTS: Array<{ id: 'LEGERE' | 'ADAPTEE' | 'LOURDE'; label: string }> = [
  { id: 'LEGERE',  label: 'Légère' },
  { id: 'ADAPTEE', label: 'Adaptée' },
  { id: 'LOURDE',  label: 'Lourde' },
];

const HEURES_CHIPS = [
  { id: 'H_MOINS_1', label: '< 1h' },
  { id: 'H_1_3',     label: '1–3h' },
  { id: 'H_3_5',     label: '3–5h' },
  { id: 'H_PLUS_5',  label: '> 5h' },
];

function ChargeSlider({
  value,
  onChange,
}: {
  value: string | null;
  onChange: (v: 'LEGERE' | 'ADAPTEE' | 'LOURDE') => void;
}) {
  const idx = CHARGE_OPTS.findIndex((o) => o.id === value);

  return (
    <View style={slider.container}>
      <View style={slider.trackRow}>
        {CHARGE_OPTS.map((opt, i) => (
          <React.Fragment key={opt.id}>
            <TouchableOpacity onPress={() => onChange(opt.id)} activeOpacity={0.7}>
              <View style={[slider.dot, { backgroundColor: idx >= i ? '#6366F1' : '#CBD5E1' }]} />
            </TouchableOpacity>
            {i < 2 && (
              <View style={[slider.segment, { backgroundColor: idx > i ? '#6366F1' : '#E2E8F0' }]} />
            )}
          </React.Fragment>
        ))}
      </View>
      <View style={slider.labels}>
        {CHARGE_OPTS.map((opt) => (
          <TouchableOpacity key={opt.id} onPress={() => onChange(opt.id)} style={slider.labelBtn}>
            <Text style={[slider.label, { color: value === opt.id ? '#6366F1' : '#64748B', fontWeight: value === opt.id ? '700' : '400' }]}>
              {opt.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>
    </View>
  );
}

export default function EvalStep3Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed =
    state.chargePerception !== null &&
    state.chargeHeuresSemaine !== null &&
    state.scoreDifficulte !== null;

  return (
    <>
      <Stack.Screen options={{ title: 'Charge', header: () => <ProgressBar currentStep={4} /> }} />
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>Charge de travail</Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Perception globale</Text>
            <ChargeSlider
              value={state.chargePerception}
              onChange={(v) => set({ chargePerception: v })}
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Heures de travail par semaine</Text>
            <ChipGroup
              chips={HEURES_CHIPS}
              selected={state.chargeHeuresSemaine ? [state.chargeHeuresSemaine] : []}
              onToggle={(id) => set({ chargeHeuresSemaine: id as any })}
              polarity="neutral"
              multiSelect={false}
            />
          </View>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.cardLabel, { color: colors.textPrimary }]}>Difficulté ressentie</Text>
            <LikertScale
              value={state.scoreDifficulte}
              onChange={(v) => set({ scoreDifficulte: v })}
              labelMin="Très facile"
              labelMax="Très difficile"
            />
          </View>
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onPrev={() => router.back()}
            onNext={() => router.push('/evaluation/step4')}
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

const slider = StyleSheet.create({
  container: { gap: 12 },
  trackRow: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 8 },
  dot: { width: 22, height: 22, borderRadius: 11 },
  segment: { flex: 1, height: 3, borderRadius: 2 },
  labels: { flexDirection: 'row', justifyContent: 'space-between' },
  labelBtn: { flex: 1, alignItems: 'center' },
  label: { fontSize: 13 },
});
