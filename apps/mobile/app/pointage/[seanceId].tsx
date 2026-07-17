import { useTheme } from '@/hooks/use-theme';
import { apiInstance, setupAxiosInterceptors } from '@/services/api';
import { Stack, useFocusEffect, useLocalSearchParams } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import QRCode from 'react-native-qrcode-svg';

// Transposition React Native de la page web-admin Presence.tsx : mêmes
// endpoints (préfixe /pointage côté student), mêmes DTO, mêmes cadences de
// rafraîchissement (token ~25 s, présence ~5 s).

type ElevePresence = {
  user_id: number;
  name: string;
  surname: string;
  statut: 'PRESENT' | 'RETARD' | 'ABSENT';
  pointe_at: string | null;
  hors_groupe?: boolean;
};

type PresenceResponse = {
  matiere: string;
  total: number;
  presents: number;
  retards: number;
  absents: number;
  eleves: ElevePresence[];
};

type TokenResponse = {
  token: string;
  code: string;
  ttl_seconds: number;
};

type OpenSeanceResponse = {
  id: number;
  code: string;
  opened_at: string;
  late_after_minutes: number;
};

const STATUT_COLOR: Record<string, string> = {
  PRESENT: '#16a34a',
  RETARD: '#d97706',
  ABSENT: '#dc2626',
};
const STATUT_LABEL: Record<string, string> = {
  PRESENT: 'Présent',
  RETARD: 'Retard',
  ABSENT: 'Absent',
};

function Metric({ label, value, color }: { label: string; value: number | string; color: string }) {
  const colors = useTheme();
  return (
    <View style={[styles.metric, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
      <Text style={[styles.metricValue, { color }]}>{value}</Text>
      <Text style={[styles.metricLabel, { color: colors.textSecondary }]}>{label}</Text>
    </View>
  );
}

export default function SeanceDetailScreen() {
  const colors = useTheme();
  const params = useLocalSearchParams<{
    seanceId: string;
    matiere?: string;
    horaire?: string;
    activated?: string;
    closed?: string;
  }>();
  const seanceId = params.seanceId;

  const [opened, setOpened] = useState(params.activated === '1');
  const [closed, setClosed] = useState(params.closed === '1');
  const [presence, setPresence] = useState<PresenceResponse | null>(null);
  const [token, setToken] = useState<TokenResponse | null>(null);
  const [erreur, setErreur] = useState('');
  const [accesRefuse, setAccesRefuse] = useState(false);
  const [actionEnCours, setActionEnCours] = useState(false);

  // refs pour les timers, nettoyés en sortie d'écran
  const presenceTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const tokenTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  const chargerPresence = useCallback(() => {
    apiInstance
      .get<PresenceResponse>(`/pointage/seance/${seanceId}/presence`)
      .then(res => {
        setPresence(res.data);
        setErreur('');
      })
      .catch(e => {
        if (e?.response?.status === 403) setAccesRefuse(true);
        else setErreur('Impossible de charger la présence.');
      });
  }, [seanceId]);

  const chargerToken = useCallback(() => {
    apiInstance
      .get<TokenResponse>(`/pointage/seance/${seanceId}/token`)
      .then(res => setToken(res.data))
      .catch(e => {
        if (e?.response?.status === 403) setAccesRefuse(true);
        else setToken(null);
      });
  }, [seanceId]);

  // Rafraîchissement périodique tant que l'écran est affiché et la séance
  // ouverte ; tout est nettoyé à la perte de focus.
  useFocusEffect(
    useCallback(() => {
      setupAxiosInterceptors();
      chargerPresence();
      if (!closed) {
        presenceTimer.current = setInterval(chargerPresence, 5_000);
      }
      return () => {
        if (presenceTimer.current) clearInterval(presenceTimer.current);
        presenceTimer.current = null;
      };
    }, [chargerPresence, closed])
  );

  useEffect(() => {
    if (opened && !closed) {
      chargerToken();
      tokenTimer.current = setInterval(chargerToken, 25_000);
    }
    return () => {
      if (tokenTimer.current) clearInterval(tokenTimer.current);
      tokenTimer.current = null;
    };
  }, [opened, closed, chargerToken]);

  const ouvrir = () => {
    setActionEnCours(true);
    apiInstance
      .post<OpenSeanceResponse>(`/pointage/seance/${seanceId}/open`, {})
      .then(() => {
        setOpened(true);
        chargerPresence();
      })
      .catch(e => {
        if (e?.response?.status === 403) setAccesRefuse(true);
        else setErreur('Erreur lors de l’ouverture de la séance.');
      })
      .finally(() => setActionEnCours(false));
  };

  const cloturer = () => {
    setActionEnCours(true);
    apiInstance
      .post(`/pointage/seance/${seanceId}/close`)
      .then(() => {
        setClosed(true);
        setToken(null);
        chargerPresence();
      })
      .catch(e => {
        if (e?.response?.status === 403) setAccesRefuse(true);
        else setErreur('Erreur lors de la clôture de la séance.');
      })
      .finally(() => setActionEnCours(false));
  };

  const taux = presence && presence.total > 0
    ? Math.round((presence.presents / presence.total) * 100)
    : 0;

  if (accesRefuse) {
    return (
      <View style={[styles.container, styles.centerBox, { backgroundColor: colors.pageBg }]}>
        <Stack.Screen options={{ title: params.matiere || 'Séance' }} />
        <Text style={styles.errorText}>
          Accès refusé : cette séance n’est pas rattachée à votre compte.
        </Text>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
      <Stack.Screen options={{ title: params.matiere || 'Séance' }} />

      <FlatList
        data={presence?.eleves ?? []}
        keyExtractor={e => String(e.user_id)}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <View style={styles.header}>
            {params.horaire ? (
              <Text style={[styles.horaire, { color: colors.textSecondary }]}>{params.horaire}</Text>
            ) : null}

            {erreur ? <Text style={styles.errorText}>{erreur}</Text> : null}

            {closed && (
              <View style={[styles.closedBanner, { borderColor: colors.cardBorder, backgroundColor: colors.cardBg }]}>
                <Text style={[styles.closedText, { color: colors.textSecondary }]}>
                  Séance clôturée — la liste ci-dessous est figée.
                </Text>
              </View>
            )}

            {!opened && !closed && (
              <TouchableOpacity
                style={[styles.mainBtn, { backgroundColor: colors.tint }, actionEnCours && styles.btnDisabled]}
                onPress={ouvrir}
                disabled={actionEnCours}
              >
                {actionEnCours ? (
                  <ActivityIndicator color={colors.background} />
                ) : (
                  <Text style={[styles.mainBtnText, { color: colors.background }]}>Ouvrir le pointage</Text>
                )}
              </TouchableOpacity>
            )}

            {opened && !closed && token && (
              <View style={[styles.qrCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                <View style={styles.qrBox}>
                  <QRCode value={token.token} size={220} backgroundColor="#ffffff" color="#000000" quietZone={10} />
                </View>
                <Text style={[styles.qrHint, { color: colors.textSecondary }]}>
                  Valide {token.ttl_seconds}s · se rafraîchit automatiquement
                </Text>
                <TouchableOpacity
                  style={[styles.closeBtn, actionEnCours && styles.btnDisabled]}
                  onPress={cloturer}
                  disabled={actionEnCours}
                >
                  <Text style={styles.closeBtnText}>Clôturer la séance</Text>
                </TouchableOpacity>
              </View>
            )}

            {presence ? (
              <View style={styles.metrics}>
                <Metric label="Présents" value={presence.presents} color="#16a34a" />
                <Metric label="Retards" value={presence.retards} color="#d97706" />
                <Metric label="Absents" value={presence.absents} color="#dc2626" />
                <Metric label="Présence" value={`${taux}%`} color={colors.textPrimary} />
              </View>
            ) : (
              <View style={styles.centerBox}>
                <ActivityIndicator size="large" color={colors.tint} />
              </View>
            )}
          </View>
        }
        renderItem={({ item }) => (
          <View style={[styles.eleveRow, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <View style={[styles.statutDot, { backgroundColor: STATUT_COLOR[item.statut] ?? '#6b7280' }]} />
            <View style={styles.eleveInfo}>
              <Text style={[styles.eleveNom, { color: colors.textPrimary }]}>
                {item.surname} {item.name}
                {item.hors_groupe ? (
                  <Text style={[styles.horsGroupe, { color: colors.textSecondary }]}> (H.G.)</Text>
                ) : null}
              </Text>
              <Text style={[styles.eleveStatut, { color: STATUT_COLOR[item.statut] ?? colors.textSecondary }]}>
                {STATUT_LABEL[item.statut] ?? item.statut}
                {item.pointe_at
                  ? ` · ${new Date(item.pointe_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}`
                  : ''}
              </Text>
            </View>
          </View>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  list: {
    padding: 16,
    gap: 8,
  },
  header: {
    gap: 12,
    marginBottom: 8,
  },
  horaire: {
    fontSize: 14,
    fontWeight: '600',
  },
  mainBtn: {
    paddingVertical: 14,
    borderRadius: 8,
    alignItems: 'center',
  },
  mainBtnText: {
    fontSize: 15,
    fontWeight: '600',
  },
  btnDisabled: {
    opacity: 0.6,
  },
  qrCard: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 20,
    alignItems: 'center',
    gap: 6,
  },
  qrBox: {
    backgroundColor: '#ffffff',
    borderRadius: 8,
    padding: 8,
  },
  qrHint: {
    fontSize: 12,
  },
  closeBtn: {
    marginTop: 12,
    borderWidth: 1,
    borderColor: '#d32f2f',
    borderRadius: 8,
    paddingHorizontal: 20,
    paddingVertical: 10,
  },
  closeBtnText: {
    color: '#d32f2f',
    fontSize: 14,
    fontWeight: '600',
  },
  closedBanner: {
    borderRadius: 8,
    borderWidth: 1,
    padding: 12,
  },
  closedText: {
    fontSize: 13,
  },
  metrics: {
    flexDirection: 'row',
    gap: 8,
  },
  metric: {
    flex: 1,
    borderRadius: 10,
    borderWidth: 1,
    paddingVertical: 10,
    alignItems: 'center',
  },
  metricValue: {
    fontSize: 20,
    fontWeight: '700',
  },
  metricLabel: {
    fontSize: 11,
  },
  eleveRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 10,
    borderWidth: 1,
    padding: 12,
    gap: 12,
  },
  statutDot: {
    width: 12,
    height: 12,
    borderRadius: 6,
  },
  eleveInfo: {
    flex: 1,
    gap: 2,
  },
  eleveNom: {
    fontSize: 15,
    fontWeight: '500',
  },
  horsGroupe: {
    fontStyle: 'italic',
    fontWeight: '400',
    fontSize: 13,
  },
  eleveStatut: {
    fontSize: 13,
  },
  centerBox: {
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  errorText: {
    color: '#d32f2f',
    fontSize: 14,
    textAlign: 'center',
  },
});
