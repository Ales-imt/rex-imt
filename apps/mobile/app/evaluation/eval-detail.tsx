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
import { Circle, G, Path, Svg } from 'react-native-svg';
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

const SECTION_COLORS: Record<string, string> = {
  'Contexte':             '#0EA5E9',
  'Pédagogie':            '#6366F1',
  'Contenu':              '#0EA5E9',
  'Charge de travail':    '#F97316',
  'Supports':             '#8B5CF6',
  'Ambiance':             '#22C55E',
  'Recommandation (NPS)': '#6366F1',
};

function npsColor(nps: number): string {
  if (nps <= 6) return '#EF4444';
  if (nps <= 8) return '#F97316';
  return '#22C55E';
}

function StarRow({ score, max = 5 }: { score: number; max?: number }) {
  return (
    <View style={{ flexDirection: 'row', gap: 2 }}>
      {Array.from({ length: max }, (_, i) => (
        <Text key={i} style={{ fontSize: 16, color: i < score ? '#F59E0B' : '#D1D5DB' }}>
          {i < score ? '★' : '☆'}
        </Text>
      ))}
    </View>
  );
}

function NpsGauge({ nps, colors }: { nps: number; colors: any }) {
  const r = 50;
  const cx = 60;
  const cy = 60;
  const startAngle = Math.PI;
  const endAngle = 0;
  const fillAngle = Math.PI - (nps / 10) * Math.PI;

  function polarToCartesian(angle: number) {
    return {
      x: cx + r * Math.cos(angle),
      y: cy - r * Math.sin(angle),
    };
  }

  const bgStart = polarToCartesian(startAngle);
  const bgEnd = polarToCartesian(endAngle);
  const fgStart = polarToCartesian(startAngle);
  const fgEnd = polarToCartesian(fillAngle);
  const fillPct = nps / 10;

  const color = npsColor(nps);

  return (
    <View style={{ alignItems: 'center', paddingVertical: 4 }}>
      <Svg width={120} height={70} viewBox="0 0 120 70">
        {/* Background arc */}
        <Path
          d={`M ${bgStart.x} ${bgStart.y} A ${r} ${r} 0 0 1 ${bgEnd.x} ${bgEnd.y}`}
          stroke="#E2E8F0"
          strokeWidth={10}
          fill="none"
          strokeLinecap="round"
        />
        {/* Fill arc */}
        {nps > 0 && (
          <Path
            d={`M ${fgStart.x} ${fgStart.y} A ${r} ${r} 0 ${fillPct > 0.5 ? 0 : 0} 1 ${fgEnd.x} ${fgEnd.y}`}
            stroke={color}
            strokeWidth={10}
            fill="none"
            strokeLinecap="round"
          />
        )}
      </Svg>
      <Text style={{ fontSize: 28, fontWeight: '800', color, marginTop: -28 }}>
        {nps}<Text style={{ fontSize: 14, fontWeight: '400', color: colors.textSecondary }}>/10</Text>
      </Text>
    </View>
  );
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
          headers: { 'X-Pseudo': pseudo },
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
        headers: { 'X-Pseudo': pseudo },
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
                <StarRowItem label="Qualité globale" score={session.score_peda_global} colors={colors} />
                <StarRowItem label="Clarté" score={session.score_peda_clarte} colors={colors} />
              </Section>

              <Section title="Contenu" colors={colors}>
                <StarRowItem label="Adéquation" score={session.score_contenu_adequation} colors={colors} />
              </Section>

              <Section title="Charge de travail" colors={colors}>
                <Row label="Perception" value={CHARGE_LABEL[session.charge_perception] ?? session.charge_perception} colors={colors} />
                <Row label="Heures / semaine" value={HEURES_LABEL[session.charge_heures_semaine] ?? session.charge_heures_semaine} colors={colors} />
                <StarRowItem label="Difficulté" score={session.score_difficulte} colors={colors} />
              </Section>

              <Section title="Supports" colors={colors}>
                <StarRowItem label="Note" score={session.score_supports} colors={colors} />
              </Section>

              <Section title="Ambiance" colors={colors}>
                <StarRowItem label="Note" score={session.score_ambiance} colors={colors} />
              </Section>

              <Section title="Recommandation (NPS)" colors={colors}>
                <NpsGauge nps={session.nps} colors={colors} />
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
  const accent = SECTION_COLORS[title] ?? '#6366F1';
  return (
    <View style={[
      styles.section,
      { backgroundColor: colors.cardBg, borderColor: colors.cardBorder, borderLeftColor: accent, borderLeftWidth: 3 },
    ]}>
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

function StarRowItem({ label, score, colors }: { label: string; score: number; colors: any }) {
  return (
    <View style={styles.row}>
      <Text style={[styles.rowLabel, { color: colors.textSecondary }]}>{label}</Text>
      <StarRow score={score} />
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
    overflow: 'hidden',
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
