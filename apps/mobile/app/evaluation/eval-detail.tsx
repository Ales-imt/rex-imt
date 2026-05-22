import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { getOrCreatePseudo } from '@/services/tokens';

type SessionDetail = {
  session_id: string;
  format_suivi: string;
  tendance_semestre: string;
  assiduite: string;
  score_peda_global: number;
  score_peda_clarte: number;
  score_contenu_adequation: number;
  charge_perception: string;
  charge_heures_semaine: string;
  score_difficulte: number;
  score_supports: number;
  score_ambiance: number;
  nps: number;
};

const FORMAT_LABEL: Record<string, string> = {
  PRESENTIEL: 'Présentiel',
  DISTANCIEL: 'Distanciel',
  HYBRIDE: 'Hybride',
};
const TENDANCE_LABEL: Record<string, string> = {
  PROGRES: 'En progrès',
  STABLE: 'Stable',
  BAISSE: 'En baisse',
};
const ASSIDUITE_LABEL: Record<string, string> = {
  TOUTES: 'Toutes les séances',
  QUELQUES_ABSENCES: 'Quelques absences',
  BEAUCOUP: "Beaucoup d'absences",
};
const CHARGE_LABEL: Record<string, string> = {
  LEGERE: 'Légère',
  ADAPTEE: 'Adaptée',
  LOURDE: 'Lourde',
};
const HEURES_LABEL: Record<string, string> = {
  H_MOINS_1: '< 1h / semaine',
  H_1_3: '1–3h / semaine',
  H_3_5: '3–5h / semaine',
  H_PLUS_5: '> 5h / semaine',
};

function stars(score: number, max = 5): string {
  return '★'.repeat(score) + '☆'.repeat(Math.max(0, max - score));
}

function npsColor(nps: number): string {
  if (nps <= 6) return '#EF4444';
  if (nps <= 8) return '#F97316';
  return '#22C55E';
}

export default function EvalDetailScreen() {
  const colors = useTheme();
  const insets = useSafeAreaInsets();
  const { matiereId, nom } = useLocalSearchParams<{ matiereId: string; nom: string }>();

  const [session, setSession] = useState<SessionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    getOrCreatePseudo()
      .then((pseudo) =>
        apiInstance.get<SessionDetail>('/evaluation/session', {
          headers: { 'X-Anon-Id': pseudo },
          params: { matiere_id: matiereId },
        })
      )
      .then(({ data }) => { if (active) setSession(data); })
      .catch(() => { if (active) setError('Impossible de charger le détail.'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [matiereId]);

  function confirmDelete() {
    Alert.alert(
      'Supprimer cette évaluation',
      'Cette action est irréversible. Votre avis sera définitivement supprimé.',
      [
        { text: 'Annuler', style: 'cancel' },
        { text: 'Supprimer', style: 'destructive', onPress: doDelete },
      ]
    );
  }

  async function doDelete() {
    if (!session || deleting) return;
    setDeleting(true);
    try {
      const pseudo = await getOrCreatePseudo();
      await apiInstance.delete(`/evaluation/session/${session.session_id}`, {
        headers: { 'X-Anon-Id': pseudo },
      });
      router.replace('/evaluation');
    } catch {
      Alert.alert('Erreur', "Impossible de supprimer l'évaluation.");
      setDeleting(false);
    }
  }

  return (
    <>
      <Stack.Screen options={{ title: nom ?? 'Détail évaluation' }} />
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        {loading ? (
          <ActivityIndicator color="#6366F1" style={styles.centered} size="large" />
        ) : error ? (
          <Text style={[styles.errorText, { color: 'red' }]}>{error}</Text>
        ) : session ? (
          <>
            <ScrollView contentContainerStyle={styles.scroll}>
              <Section title="Contexte" colors={colors}>
                <Row label="Format" value={FORMAT_LABEL[session.format_suivi] ?? session.format_suivi} colors={colors} />
                <Row label="Tendance" value={TENDANCE_LABEL[session.tendance_semestre] ?? session.tendance_semestre} colors={colors} />
                <Row label="Assiduité" value={ASSIDUITE_LABEL[session.assiduite] ?? session.assiduite} colors={colors} />
              </Section>

              <Section title="Pédagogie" colors={colors}>
                <Row label="Qualité globale" value={stars(session.score_peda_global)} colors={colors} />
                <Row label="Clarté" value={stars(session.score_peda_clarte)} colors={colors} />
              </Section>

              <Section title="Contenu" colors={colors}>
                <Row label="Adéquation" value={stars(session.score_contenu_adequation)} colors={colors} />
              </Section>

              <Section title="Charge de travail" colors={colors}>
                <Row label="Perception" value={CHARGE_LABEL[session.charge_perception] ?? session.charge_perception} colors={colors} />
                <Row label="Heures / semaine" value={HEURES_LABEL[session.charge_heures_semaine] ?? session.charge_heures_semaine} colors={colors} />
                <Row label="Difficulté" value={stars(session.score_difficulte)} colors={colors} />
              </Section>

              <Section title="Supports" colors={colors}>
                <Row label="Note" value={stars(session.score_supports)} colors={colors} />
              </Section>

              <Section title="Ambiance" colors={colors}>
                <Row label="Note" value={stars(session.score_ambiance)} colors={colors} />
              </Section>

              <Section title="Recommandation (NPS)" colors={colors}>
                <View style={styles.npsContainer}>
                  <Text style={[styles.npsScore, { color: npsColor(session.nps) }]}>
                    {session.nps}
                    <Text style={[styles.npsMax, { color: colors.textSecondary }]}>/10</Text>
                  </Text>
                </View>
              </Section>
            </ScrollView>

            <View style={[styles.footer, { paddingBottom: insets.bottom + 16 }]}>
              <TouchableOpacity
                style={[styles.deleteBtn, deleting && styles.deleteBtnDisabled]}
                onPress={confirmDelete}
                activeOpacity={0.8}
                disabled={deleting}
              >
                {deleting ? (
                  <ActivityIndicator color="#fff" size="small" />
                ) : (
                  <Text style={styles.deleteBtnText}>Supprimer cette évaluation</Text>
                )}
              </TouchableOpacity>
            </View>
          </>
        ) : null}
      </View>
    </>
  );
}

function Section({ title, children, colors }: { title: string; children: React.ReactNode; colors: any }) {
  return (
    <View style={[styles.section, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
      <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>{title.toUpperCase()}</Text>
      {children}
    </View>
  );
}

function Row({ label, value, colors }: { label: string; value: string; colors: any }) {
  return (
    <View style={styles.row}>
      <Text style={[styles.rowLabel, { color: colors.textSecondary }]}>{label}</Text>
      <Text style={[styles.rowValue, { color: colors.textPrimary }]}>{value}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { marginTop: 60 },
  errorText: { textAlign: 'center', marginTop: 40, fontSize: 14 },
  scroll: { padding: 16, gap: 12, paddingBottom: 8 },
  section: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    gap: 10,
  },
  sectionTitle: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.8,
    marginBottom: 2,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  rowLabel: { fontSize: 14 },
  rowValue: { fontSize: 14, fontWeight: '600' },
  npsContainer: { alignItems: 'center', paddingVertical: 4 },
  npsScore: { fontSize: 36, fontWeight: '800' },
  npsMax: { fontSize: 18, fontWeight: '400' },
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
  },
  deleteBtn: {
    backgroundColor: '#EF4444',
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: 'center',
  },
  deleteBtnDisabled: { opacity: 0.6 },
  deleteBtnText: { color: '#fff', fontSize: 15, fontWeight: '700' },
});
