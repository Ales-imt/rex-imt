
import { ZenBackground } from '@/components/zen-background';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { getOrCreatePseudo } from '@/services/tokens';
import { LinearGradient } from 'expo-linear-gradient';
import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';

type MatiereSummary = {
  matiereId: string;
  nom: string;
  prof: string;
  formation: string;
  fait: boolean;
};

export default function CoursListScreen() {
  const colors = useTheme();
  const { init } = useEvaluation();

  const [cours, setCours] = useState<MatiereSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useFocusEffect(
    useCallback(() => {
      let active = true;
      setLoading(true);
      setError('');

      getOrCreatePseudo()
        .then((pseudo) => apiInstance.get<MatiereSummary[]>('/evaluation/matieres', { headers: { 'X-Pseudo': pseudo } }))
        .then(({ data }) => {
          if (!active) return;
          setCours(data ?? []);
        })
        .catch(() => {
          if (active) setError('Impossible de charger les cours.');
        })
        .finally(() => {
          if (active) setLoading(false);
        });

      return () => { active = false; };
    }, [])
  );

  const done = cours.filter((c) => c.fait).length;
  const progressPct = cours.length > 0 ? (done / cours.length) * 100 : 0;

  function handlePress(c: MatiereSummary) {
    if (c.fait) {
      router.push({ pathname: '/evaluation/eval-detail', params: { matiereId: c.matiereId, nom: c.nom } });
      return;
    }
    init(c.matiereId);
    router.push('/evaluation/step0');
  }

  return (
    <>
      <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
        <ZenBackground />

        {loading ? (
          <ActivityIndicator color="#6366F1" style={styles.centered} size="large" />
        ) : error ? (
          <Text style={[styles.empty, { color: 'red' }]}>{error}</Text>
        ) : (
          <FlatList
            data={cours}
            keyExtractor={(item) => item.matiereId}
            contentContainerStyle={styles.list}
            renderItem={({ item }) => (
              <TouchableOpacity
                style={[
                  styles.card,
                  {
                    backgroundColor: item.fait ? colors.evalDoneBg : colors.evalTodoBg,
                    borderColor: item.fait ? colors.evalDoneBorder : colors.evalTodoBorder,
                  },
                ]}
                onPress={() => handlePress(item)}
                activeOpacity={0.7}
              >
                <View style={styles.cardBody}>
                  <Text style={[styles.coursName, { color: colors.textPrimary }]} numberOfLines={2}>
                    {item.nom}
                  </Text>
                  <Text style={[styles.coursMeta, { color: colors.textSecondary }]}>
                    {[item.prof, item.formation].filter(Boolean).join(' · ')}
                  </Text>
                </View>
                {item.fait ? (
                  <View style={styles.badgeDone}>
                    <Text style={styles.badgeDoneText}>Fait ✓</Text>
                  </View>
                ) : (
                  <View style={styles.badgeTodo}>
                    <Text style={styles.badgeTodoText}>À faire →</Text>
                  </View>
                )}
              </TouchableOpacity>
            )}
            ListEmptyComponent={
              <Text style={[styles.empty, { color: colors.textSecondary }]}>
                Aucun cours sur cette année scolaire.
              </Text>
            }
          />
        )}

        {!loading && !error && cours.length > 0 && (
          <View style={[styles.footer, { backgroundColor: colors.cardBg, borderTopColor: colors.cardBorder }]}>
            <View style={styles.progressTrack}>
              <LinearGradient
                colors={['#6366F1', '#8B5CF6', '#22C55E']}
                start={{ x: 0, y: 0 }}
                end={{ x: 1, y: 0 }}
                style={[styles.progressFill, { width: `${progressPct}%` }]}
              />
              {progressPct > 0 && progressPct < 100 && (
                <View style={[styles.progressDot, { left: `${progressPct}%` as any }]} />
              )}
            </View>
            <Text style={[styles.progressLabel, { color: colors.textSecondary }]}>
              {done}/{cours.length} cours évalué{done > 1 ? 's' : ''}
            </Text>
          </View>
        )}
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { marginTop: 60 },
  list: { padding: 16, gap: 12 },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 12,
    borderWidth: 1,
    padding: 14,
    gap: 12,
    boxShadow: '0 2px 4px rgba(0,0,0,0.06)',
    elevation: 2,
  },
  cardBody: { flex: 1, gap: 3 },
  coursName: { fontSize: 15, fontWeight: '600' },
  coursMeta: { fontSize: 12 },
  badgeTodo: {
    backgroundColor: '#6366F1',
    borderRadius: 20,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  badgeTodoText: { fontSize: 12, fontWeight: '600', color: '#ffffff' },
  badgeDone: {
    backgroundColor: '#16A34A',
    borderRadius: 20,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  badgeDoneText: { fontSize: 12, fontWeight: '600', color: '#ffffff' },
  empty: { textAlign: 'center', marginTop: 40, fontSize: 14, fontStyle: 'italic' },
  footer: { padding: 16, borderTopWidth: 1, gap: 8 },
  progressTrack: {
    height: 10,
    borderRadius: 5,
    backgroundColor: '#E2E8F0',
    overflow: 'visible',
    position: 'relative',
  },
  progressFill: { height: 10, borderRadius: 5 },
  progressDot: {
    position: 'absolute',
    top: -3,
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: '#ffffff',
    borderWidth: 2,
    borderColor: '#6366F1',
    marginLeft: -8,
  },
  progressLabel: { fontSize: 13, textAlign: 'center' },
});
