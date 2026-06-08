
import { ZenBackground } from '@/components/zen-background';
import { useEvaluation } from '@/hooks/use-evaluation';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { getOrCreatePseudo } from '@/services/tokens';
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
                  { backgroundColor: colors.cardBg, borderColor: colors.cardBorder },
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
                <View style={[styles.badge, { backgroundColor: item.fait ? '#DCFCE7' : '#EEF2FF' }]}>
                  <Text style={[styles.badgeText, { color: item.fait ? '#16A34A' : '#6366F1' }]}>
                    {item.fait ? 'Fait ✓' : 'À faire'}
                  </Text>
                </View>
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
            <View style={styles.progressBar}>
              <View style={[styles.progressFill, { width: `${(done / cours.length) * 100}%` }]} />
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
    shadowColor: '#000',
    shadowOpacity: 0.06,
    shadowOffset: { width: 0, height: 2 },
    shadowRadius: 4,
    elevation: 2,
  },
  cardDone: { opacity: 0.55 },
  cardBody: { flex: 1, gap: 3 },
  coursName: { fontSize: 15, fontWeight: '600' },
  coursMeta: { fontSize: 12 },
  badge: { borderRadius: 12, paddingHorizontal: 10, paddingVertical: 5 },
  badgeText: { fontSize: 12, fontWeight: '600' },
  empty: { textAlign: 'center', marginTop: 40, fontSize: 14, fontStyle: 'italic' },
  footer: { padding: 16, borderTopWidth: 1, gap: 8 },
  progressBar: { height: 6, borderRadius: 3, backgroundColor: '#E2E8F0', overflow: 'hidden' },
  progressFill: { height: 6, borderRadius: 3, backgroundColor: '#6366F1' },
  progressLabel: { fontSize: 13, textAlign: 'center' },
});
