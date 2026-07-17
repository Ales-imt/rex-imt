import { useTheme } from '@/hooks/use-theme';
import { apiInstance, setupAxiosInterceptors } from '@/services/api';
import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';

// Aligné sur seanceJourItem du backend student (GET /pointage/seances/jour).
// Le backend a déjà filtré selon le rôle (prof → ses séances, gestionnaire →
// toutes) : l'écran affiche simplement ce qu'il reçoit.
export type SeanceJour = {
  id: number;
  matiere_id: number;
  matiere: string;
  starts_at: string;
  ends_at: string;
  salle: string;
  prof: string;
  promo: string;
  activated: boolean;
  closed_at: string | null;
};

type Etat = 'loading' | 'ok' | 'erreur';

function heure(iso: string): string {
  if (!iso) return '';
  return new Date(iso).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
}

function StatutBadge({ seance }: { seance: SeanceJour }) {
  if (seance.closed_at) {
    return <Text style={[styles.badge, styles.badgeClosed]}>Clôturée</Text>;
  }
  if (seance.activated) {
    return <Text style={[styles.badge, styles.badgeOpen]}>En cours</Text>;
  }
  return null;
}

export default function PointageScreen() {
  const colors = useTheme();
  const [etat, setEtat] = useState<Etat>('loading');
  const [seances, setSeances] = useState<SeanceJour[]>([]);
  const [refreshing, setRefreshing] = useState(false);

  const charger = useCallback((viaRefresh = false) => {
    if (viaRefresh) setRefreshing(true); else setEtat('loading');
    apiInstance
      .get<SeanceJour[]>('/pointage/seances/jour')
      .then(res => {
        setSeances(res.data ?? []);
        setEtat('ok');
      })
      .catch(() => setEtat('erreur'))
      .finally(() => setRefreshing(false));
  }, []);

  useFocusEffect(
    useCallback(() => {
      setupAxiosInterceptors();
      charger();
    }, [charger])
  );

  const ouvrirDetail = (s: SeanceJour) => {
    router.push({
      pathname: '/pointage/[seanceId]',
      params: {
        seanceId: String(s.id),
        matiere: s.matiere,
        horaire: `${heure(s.starts_at)} – ${heure(s.ends_at)}`,
        activated: s.activated ? '1' : '0',
        closed: s.closed_at ? '1' : '0',
      },
    });
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
      <Text style={[styles.title, { color: colors.textPrimary }]}>Cours du jour</Text>

      {etat === 'loading' && (
        <View style={styles.centerBox}>
          <ActivityIndicator size="large" color={colors.tint} />
        </View>
      )}

      {etat === 'erreur' && (
        <View style={styles.centerBox}>
          <Text style={styles.errorText}>Impossible de charger les cours du jour.</Text>
          <TouchableOpacity style={[styles.retryBtn, { backgroundColor: colors.tint }]} onPress={() => charger()}>
            <Text style={[styles.retryBtnText, { color: colors.background }]}>Réessayer</Text>
          </TouchableOpacity>
        </View>
      )}

      {etat === 'ok' && seances.length === 0 && (
        <View style={styles.centerBox}>
          <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
            Aucun cours aujourd’hui.
          </Text>
        </View>
      )}

      {etat === 'ok' && seances.length > 0 && (
        <FlatList
          data={seances}
          keyExtractor={s => String(s.id)}
          contentContainerStyle={styles.list}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={() => charger(true)} tintColor={colors.tint} />
          }
          renderItem={({ item }) => (
            <TouchableOpacity
              style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}
              onPress={() => ouvrirDetail(item)}
              activeOpacity={0.7}
            >
              <View style={styles.cardHeader}>
                <Text style={[styles.horaire, { color: colors.tint }]}>
                  {heure(item.starts_at)} – {heure(item.ends_at)}
                </Text>
                <StatutBadge seance={item} />
              </View>
              <Text style={[styles.matiere, { color: colors.textPrimary }]}>{item.matiere}</Text>
              <Text style={[styles.detail, { color: colors.textSecondary }]}>
                {[item.salle, item.prof, item.promo].filter(Boolean).join(' · ')}
              </Text>
            </TouchableOpacity>
          )}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  title: {
    fontSize: 18,
    fontWeight: '700',
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 8,
  },
  list: {
    padding: 16,
    paddingTop: 4,
    gap: 10,
  },
  card: {
    borderRadius: 10,
    borderWidth: 1,
    padding: 14,
    gap: 4,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  horaire: {
    fontSize: 14,
    fontWeight: '700',
  },
  matiere: {
    fontSize: 16,
    fontWeight: '600',
  },
  detail: {
    fontSize: 13,
  },
  badge: {
    fontSize: 12,
    fontWeight: '600',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 6,
    overflow: 'hidden',
  },
  badgeOpen: {
    color: '#16a34a',
    backgroundColor: 'rgba(22,163,74,0.12)',
  },
  badgeClosed: {
    color: '#6b7280',
    backgroundColor: 'rgba(107,114,128,0.15)',
  },
  centerBox: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
    gap: 12,
  },
  errorText: {
    color: '#d32f2f',
    fontSize: 14,
    textAlign: 'center',
  },
  emptyText: {
    fontSize: 15,
  },
  retryBtn: {
    paddingHorizontal: 20,
    paddingVertical: 10,
    borderRadius: 8,
  },
  retryBtnText: {
    fontSize: 14,
    fontWeight: '600',
  },
});
