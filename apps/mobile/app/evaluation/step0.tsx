import { ChipGroup } from '@/components/eval/ChipGroup';
import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { Stack, router } from 'expo-router';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

const FORMAT_CHIPS = [
  { id: 'PRESENTIEL', label: 'Présentiel' },
  { id: 'DISTANCIEL', label: 'Distanciel' },
  { id: 'HYBRIDE',    label: 'Hybride' },
];

const TENDANCE_CHIPS = [
  { id: 'PROGRES', label: 'En progrès' },
  { id: 'STABLE',  label: 'Stable' },
  { id: 'BAISSE',  label: 'En baisse' },
];

const ASSIDUITE_CHIPS = [
  { id: 'TOUTES',            label: 'Toutes les séances' },
  { id: 'QUELQUES_ABSENCES', label: 'Quelques absences' },
  { id: 'BEAUCOUP',          label: "Beaucoup d'absences" },
];

export default function EvalStep0Screen() {
  const colors = useTheme();
  const { state, set } = useEvaluation();
  const insets = useSafeAreaInsets();

  const canProceed =
    state.formatSuivi !== null &&
    state.tendanceSemestre !== null &&
    state.assiduite !== null;

  function toggleSingle(key: 'formatSuivi' | 'tendanceSemestre' | 'assiduite', id: string) {
    set({ [key]: id } as any);
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Contexte', header: () => <ProgressBar currentStep={1} /> }} />
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>
            Quelques infos avant de commencer
          </Text>

          <Section label="Format suivi">
            <ChipGroup
              chips={FORMAT_CHIPS}
              selected={state.formatSuivi ? [state.formatSuivi] : []}
              onToggle={(id) => toggleSingle('formatSuivi', id)}
              polarity="neutral"
              multiSelect={false}
            />
          </Section>

          <Section label="Tendance sur le semestre">
            <ChipGroup
              chips={TENDANCE_CHIPS}
              selected={state.tendanceSemestre ? [state.tendanceSemestre] : []}
              onToggle={(id) => toggleSingle('tendanceSemestre', id)}
              polarity="neutral"
              multiSelect={false}
            />
          </Section>

          <Section label="Assiduité">
            <ChipGroup
              chips={ASSIDUITE_CHIPS}
              selected={state.assiduite ? [state.assiduite] : []}
              onToggle={(id) => toggleSingle('assiduite', id)}
              polarity="neutral"
              multiSelect={false}
            />
          </Section>
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          <NavButtons
            onNext={() => router.replace('/evaluation/step1')}
            nextDisabled={!canProceed}
            nextLabel="Commencer →"
          />
        </View>
      </View>
    </>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  const colors = useTheme();
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>{label}</Text>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  scroll: { padding: 20, gap: 4 },
  title: { fontSize: 20, fontWeight: '700', marginBottom: 12 },
  section: { gap: 10, marginBottom: 20 },
  sectionLabel: { fontSize: 13, fontWeight: '600', textTransform: 'uppercase', letterSpacing: 0.5 },
});
