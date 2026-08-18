import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { Stack } from 'expo-router';
import { useEffect, useState } from 'react';
import {
    ActivityIndicator,
    ScrollView,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';

type Matiere = {
    nom: string;
    note: number;
    coefficient: number;
    date: string;
    commentaire?: string;
};

type UE = {
    nom: string;
    score: number;
    coefficient: number;
    matieres: Matiere[];
};

type Periode = {
    nom: string;
    gpa: number;
    ues: UE[];
};

function periodeLabel(nom: string): string {
    const parts = nom.split('_');
    return parts[parts.length - 1] ?? nom;
}

type Colors = ReturnType<typeof useTheme>;

function MatiereRow({ m, colors }: { m: Matiere; colors: Colors }) {
    return (
        <View style={[styles.matiereRow, { borderTopColor: colors.dividerLine }]}>
            <View style={styles.matiereLeft}>
                <Text style={[styles.matiereNom, { color: colors.textPrimary }]}>{m.nom}</Text>
                <Text style={[styles.matiereDate, { color: colors.textSecondary }]}>{m.date}</Text>
            </View>
            <Text style={[styles.matiereNote, { color: colors.textPrimary }]}>
                {m.note}
                <Text style={{ color: colors.textSecondary }}>/20</Text>
            </Text>
        </View>
    );
}

function UECard({ ue, colors }: { ue: UE; colors: Colors }) {
    const [open, setOpen] = useState(false);
    const hasScore = ue.coefficient > 0;

    return (
        <View style={[styles.ueCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <TouchableOpacity onPress={() => setOpen(o => !o)} activeOpacity={0.8} style={styles.ueHeader}>
                <Text style={[styles.ueNom, { color: colors.textPrimary }]} numberOfLines={2}>
                    {ue.nom}
                </Text>
                <View style={styles.ueRight}>
                    {hasScore && (
                        <View style={[styles.ectsbadge, { backgroundColor: '#0a7ea4' }]}>
                            <Text style={styles.ectsbadgeText}>{ue.score}/5</Text>
                        </View>
                    )}
                    <Text style={[styles.chevron, { color: colors.textSecondary }]}>
                        {open ? '▲' : '▼'}
                    </Text>
                </View>
            </TouchableOpacity>

            {open && (ue.matieres ?? []).length === 0 && (
                <Text style={[styles.emptyMatieres, { color: colors.textSecondary }]}>
                    Aucune matière
                </Text>
            )}
            {open && (ue.matieres ?? []).map((m, i) => (
                <MatiereRow key={i} m={m} colors={colors} />
            ))}
        </View>
    );
}

function PeriodeSection({ periode, colors }: { periode: Periode; colors: Colors }) {
    const [open, setOpen] = useState(false);
    const label = periodeLabel(periode.nom);

    return (
        <View style={styles.periodeSection}>
            <TouchableOpacity
                onPress={() => setOpen(o => !o)}
                activeOpacity={0.85}
                style={[styles.periodeHeader, { backgroundColor: '#0a7ea4' }]}
            >
                <Text style={styles.periodeLabel}>{label}</Text>
                <View style={styles.periodeRight}>
                    {periode.gpa > 0 && (
                        <Text style={styles.periodeGpa}>GPA {periode.gpa.toFixed(2)}/5</Text>
                    )}
                    <Text style={styles.periodeChevron}>{open ? '▲' : '▼'}</Text>
                </View>
            </TouchableOpacity>

            {open && (
                <View style={styles.periodeContent}>
                    {periode.ues.map((ue, i) => (
                        <UECard key={i} ue={ue} colors={colors} />
                    ))}
                </View>
            )}
        </View>
    );
}

export const NotesScreen = () => {
    const colors = useTheme();
    const navMenu = useNavMenu('notes');
    const [periodes, setPeriodes] = useState<Periode[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    useEffect(() => {
        apiInstance
            .get<Periode[]>('/note')
            .then(res => setPeriodes((res.data ?? []).reverse()))
            .catch(e => setError(e?.response?.data?.message ?? e?.message ?? 'Erreur réseau'))
            .finally(() => setLoading(false));
    }, []);

    return (
        <>
            <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />
            <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
                <ZenBackground />
                {loading ? (
                    <ActivityIndicator size="large" color={colors.tint} style={styles.loader} />
                ) : error ? (
                    <View style={styles.errorBox}>
                        <Text style={styles.errorText}>{error}</Text>
                    </View>
                ) : periodes.length === 0 ? (
                    <View style={styles.errorBox}>
                        <Text style={[styles.errorText, { color: colors.textSecondary }]}>Pas de notes</Text>
                    </View>
                ) : (
                    <ScrollView contentContainerStyle={styles.scrollContent}>
                        {periodes.map((p, i) => (
                            <PeriodeSection key={i} periode={p} colors={colors} />
                        ))}
                    </ScrollView>
                )}
            </View>
        </>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
    },
    loader: {
        flex: 1,
    },
    errorBox: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 24,
    },
    errorText: {
        color: '#d32f2f',
        fontSize: 14,
        textAlign: 'center',
    },
    scrollContent: {
        padding: 12,
        gap: 8,
    },
    // Periode
    periodeSection: {
        borderRadius: 10,
        overflow: 'hidden',
    },
    periodeHeader: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: 16,
        paddingVertical: 14,
    },
    periodeLabel: {
        color: '#fff',
        fontSize: 16,
        fontWeight: '700',
    },
    periodeRight: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 10,
    },
    periodeGpa: {
        color: 'rgba(255,255,255,0.9)',
        fontSize: 13,
        fontWeight: '600',
    },
    periodeChevron: {
        color: '#fff',
        fontSize: 11,
    },
    periodeContent: {
        gap: 6,
        paddingTop: 6,
    },
    // UE
    ueCard: {
        borderRadius: 8,
        borderWidth: 1,
        overflow: 'hidden',
    },
    ueHeader: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: 14,
        paddingVertical: 12,
        gap: 8,
    },
    ueNom: {
        flex: 1,
        fontSize: 14,
        fontWeight: '500',
    },
    ueRight: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    ectsbadge: {
        borderRadius: 12,
        paddingHorizontal: 8,
        paddingVertical: 2,
    },
    ectsbadgeText: {
        color: '#fff',
        fontSize: 12,
        fontWeight: '600',
    },
    chevron: {
        fontSize: 10,
    },
    emptyMatieres: {
        paddingHorizontal: 14,
        paddingBottom: 12,
        fontSize: 13,
        fontStyle: 'italic',
    },
    // Matiere
    matiereRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: 14,
        paddingVertical: 10,
        borderTopWidth: 1,
        gap: 8,
    },
    matiereLeft: {
        flex: 1,
        gap: 2,
    },
    matiereNom: {
        fontSize: 13,
    },
    matiereDate: {
        fontSize: 11,
    },
    matiereNote: {
        fontSize: 14,
        fontWeight: '600',
    },
});

export default NotesScreen;
