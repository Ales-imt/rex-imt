import { NavButtons } from '@/components/eval/NavButtons';
import { ProgressBar } from '@/components/eval/ProgressBar';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { getOrCreatePseudo } from '@/services/tokens';
import { Stack, router } from 'expo-router';
import { useState } from 'react';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';

const NPS_COLORS: Record<string, string> = Object.fromEntries([
  ...[0, 1, 2, 3, 4, 5, 6].map((n) => [n, '#EF4444']),
  ...[7, 8].map((n) => [n, '#F97316']),
  ...[9, 10].map((n) => [n, '#22C55E']),
]);

export default function EvalStep6Screen() {
  const colors = useTheme();
  const { state, set, reset } = useEvaluation();
  const [loading, setLoading] = useState(false);
  const insets = useSafeAreaInsets();

  const canSubmit = state.nps !== null;

  async function handleSubmit() {
    if (!canSubmit || loading) return;
    setLoading(true);
    try {
      const pseudo = await getOrCreatePseudo();
      await apiInstance.post(
        '/evaluation',
        {
          matiere_id:              state.matiereId,
          format_suivi:            state.formatSuivi,
          tendance_semestre:       state.tendanceSemestre,
          assiduite:               state.assiduite,
          score_peda_global:       state.scorePedaGlobal,
          score_peda_clarte:       state.scorePedaClarte,
          score_contenu_adequation: state.scoreContenuAdequation,
          chips_contenu:           state.chipsContenu,
          charge_perception:       state.chargePerception,
          charge_heures_semaine:   state.chargeHeuresSemaine,
          score_difficulte:        state.scoreDifficulte,
          score_supports:          state.scoreSupports,
          chips_supports:          state.chipsSupports,
          verbatim_supports:       state.verbatimSupports,
          score_ambiance:          state.scoreAmbiance,
          chips_ambiance:          state.chipsAmbiance,
          verbatim_ambiance:       state.verbatimAmbiance,
          nps:                     state.nps,
          verbatim_nps:            state.verbatimNps,
        },
        { headers: { 'X-Pseudo': pseudo } }
      );
      reset();
      router.replace('/evaluation');
    } catch (err: any) {
      const msg = err?.response?.data?.error ?? 'Une erreur est survenue. Réessayez.';
      Alert.alert('Erreur', msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <Stack.Screen options={{ title: 'Recommandation', header: () => <ProgressBar currentStep={7} /> }} />
      <KeyboardAvoidingView
        style={[styles.container, { backgroundColor: colors.pageBg }]}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={[styles.title, { color: colors.textPrimary }]}>
            Recommanderiez-vous ce cours ?
          </Text>

          <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <View style={styles.npsGrid}>
              {Array.from({ length: 11 }, (_, n) => {
                const active = state.nps === n;
                const bg = NPS_COLORS[n];
                return (
                  <TouchableOpacity
                    key={n}
                    onPress={() => set({ nps: n })}
                    activeOpacity={0.7}
                    style={[
                      styles.npsBtn,
                      { backgroundColor: active ? bg : 'transparent', borderColor: active ? bg : '#CBD5E1' },
                    ]}
                  >
                    <Text style={[styles.npsBtnText, { color: active ? '#fff' : '#64748B' }]}>{n}</Text>
                  </TouchableOpacity>
                );
              })}
            </View>
            <View style={styles.npsLegend}>
              <Text style={[styles.legendText, { color: colors.textSecondary }]}>Pas du tout</Text>
              <Text style={[styles.legendText, { color: colors.textSecondary }]}>Absolument</Text>
            </View>
          </View>

          <TextInput
            style={[styles.textInput, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder, color: colors.textPrimary }]}
            placeholder="Pourquoi ? (optionnel)"
            placeholderTextColor={colors.inputPlaceholder}
            value={state.verbatimNps}
            onChangeText={(t) => set({ verbatimNps: t })}
            multiline
            maxLength={300}
            textAlignVertical="top"
          />
        </ScrollView>

        <View style={{ paddingBottom: insets.bottom }}>
          {loading ? (
            <View style={styles.loadingBar}>
              <ActivityIndicator color="#6366F1" />
            </View>
          ) : (
            <NavButtons
              onPrev={() => router.back()}
              onSubmit={handleSubmit}
              nextDisabled={!canSubmit}
            />
          )}
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
    shadowColor: '#000',
    shadowOpacity: 0.04,
    shadowOffset: { width: 0, height: 1 },
    shadowRadius: 3,
    elevation: 1,
  },
  npsGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, justifyContent: 'center' },
  npsBtn: {
    width: 48,
    height: 48,
    borderRadius: 10,
    borderWidth: 1.5,
    alignItems: 'center',
    justifyContent: 'center',
  },
  npsBtnText: { fontSize: 16, fontWeight: '700' },
  npsLegend: { flexDirection: 'row', justifyContent: 'space-between' },
  legendText: { fontSize: 11 },
  textInput: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    fontSize: 14,
    minHeight: 90,
  },
  loadingBar: { height: 72, alignItems: 'center', justifyContent: 'center' },
});
