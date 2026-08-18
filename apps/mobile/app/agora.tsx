import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useAgora } from '@/hooks/use-agora';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { Stack, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, FlatList, Modal, Pressable, StyleSheet, Text, TouchableOpacity, View } from 'react-native';

// ─── Types ────────────────────────────────────────────────────────────────────

type PostItRow = {
    id: number;
    reponse: string;
    message_modere: string | null;
    cree_le: string;
    auteur_nom: string;
    auteur_prenom: string;
    categorie: string | null;
    resume: string | null;
    sentiment: string | null;
    urgence: number | null;
};

type PostIt = {
    id: number;
    reponse: string;
    messageModere: string | null;
    auteur: string;
    tag: string;
    bgColor: string;
    rotation: number;
    timestamp: string;
    resume: string | null;
    urgence: number | null;
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

const URGENCE_COLORS: Record<number, string> = {
    1: '#dcfce7',
    2: '#dbeafe',
    3: '#fef9c3',
    4: '#fed7aa',
    5: '#fce7f3',
};

function colorFromUrgence(urgence?: number | null): string {
    if (urgence == null) return '#fef9c3';
    return URGENCE_COLORS[urgence] ?? '#fef9c3';
}

function postitRotation(id: number): number {
    return ((id * 7) % 70) / 10 - 3.5;
}

function relativeTime(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const minutes = Math.floor(diff / 60_000);
    if (minutes < 1) return "à l'instant";
    if (minutes < 60) return `il y a ${minutes} min`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `il y a ${hours} h`;
    const days = Math.floor(hours / 24);
    if (days === 1) return 'hier';
    if (days === 2) return 'avant-hier';
    return `il y a ${days} j`;
}

const TRUNCATE = 80;
function trunc(s: string): string {
    return s.length > TRUNCATE ? s.slice(0, TRUNCATE) + '…' : s;
}

function toPostIt(row: PostItRow): PostIt {
    return {
        id: row.id,
        reponse: row.reponse,
        messageModere: row.message_modere,
        auteur: `${row.auteur_nom} ${row.auteur_prenom}`.toUpperCase(),
        tag: row.categorie ?? 'Note',
        bgColor: colorFromUrgence(row.urgence),
        rotation: postitRotation(row.id),
        timestamp: relativeTime(row.cree_le),
        resume: row.resume,
        urgence: row.urgence,
    };
}

// ─── Dropdown modal ───────────────────────────────────────────────────────────

function DropdownModal<T extends string | number>({
    visible, onClose, options, selected, onSelect, label,
}: {
    visible: boolean;
    onClose: () => void;
    options: { value: T; label: string }[];
    selected: T;
    onSelect: (v: T) => void;
    label: string;
}) {
    return (
        <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
            <Pressable style={styles.overlay} onPress={onClose}>
                <Pressable style={styles.dropdown}>
                    <Text style={styles.dropdownTitle}>{label}</Text>
                    {options.map(opt => (
                        <TouchableOpacity
                            key={String(opt.value)}
                            style={[styles.dropdownItem, opt.value === selected && styles.dropdownItemActive]}
                            onPress={() => { onSelect(opt.value); onClose(); }}
                        >
                            <Text style={[styles.dropdownItemText, opt.value === selected && styles.dropdownItemTextActive]}>
                                {opt.label}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </Pressable>
            </Pressable>
        </Modal>
    );
}

// ─── Note detail modal ────────────────────────────────────────────────────────

function NoteDetailModal({ item, onClose }: { item: PostIt; onClose: () => void }) {
    return (
        <Modal visible transparent animationType="fade" onRequestClose={onClose}>
            <Pressable style={styles.overlay} onPress={onClose}>
                <Pressable style={[styles.detailBox, { backgroundColor: item.bgColor }]}>
                    <View style={styles.detailPin} />

                    <Text style={styles.detailAuteur}>{item.auteur}</Text>

                    {item.messageModere ? (
                        <View style={styles.detailSection}>
                            <Text style={styles.detailLabel}>Message modéré</Text>
                            <Text style={[styles.detailBody, styles.detailItalic]}>{item.messageModere}</Text>
                        </View>
                    ) : null}

                    <View style={styles.detailSection}>
                        <Text style={styles.detailLabel}>Réponse</Text>
                        <Text style={styles.detailBody}>{item.reponse}</Text>
                    </View>

                    {item.resume ? (
                        <View style={styles.detailSection}>
                            <Text style={styles.detailLabel}>Résumé IA</Text>
                            <Text style={styles.detailBody}>{item.resume}</Text>
                        </View>
                    ) : null}

                    <View style={styles.detailFooter}>
                        <View style={styles.chip}>
                            <Text style={styles.chipText}>{item.tag}</Text>
                        </View>
                        <Text style={styles.detailTime}>{item.timestamp}</Text>
                    </View>
                </Pressable>
            </Pressable>
        </Modal>
    );
}

// ─── Note card ────────────────────────────────────────────────────────────────

function Note({ item }: { item: PostIt }) {
    const [open, setOpen] = useState(false);

    return (
        <>
            <TouchableOpacity
                style={[styles.noteWrapper, { transform: [{ rotate: `${item.rotation}deg` }] }]}
                onPress={() => setOpen(true)}
                activeOpacity={0.85}
            >
                <View style={[styles.note, { backgroundColor: item.bgColor }]}>
                    <View style={styles.pin} />

                    <Text style={styles.noteAuteur}>{item.auteur}</Text>

                    {item.messageModere ? (
                        <Text style={styles.noteModere}>{trunc(item.messageModere)}</Text>
                    ) : null}

                    <Text style={styles.noteReponse}>{trunc(item.reponse)}</Text>

                    <View style={styles.noteFooter}>
                        <View style={styles.chip}>
                            <Text style={styles.chipText}>{item.tag}</Text>
                        </View>
                        <Text style={styles.noteTime}>{item.timestamp}</Text>
                    </View>

                </View>
            </TouchableOpacity>
            {open && <NoteDetailModal item={item} onClose={() => setOpen(false)} />}
        </>
    );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

const MONTH_OPTIONS = [1, 3, 6, 12];

export default function AgoraScreen() {
    const colors = useTheme();
    const navMenu = useNavMenu('agora');

    const { months, promotion, setMonths, setPromotion } = useAgora();
    const [promotions, setPromotions] = useState<string[]>([]);
    const [monthsOpen, setMonthsOpen] = useState(false);
    const [promoOpen, setPromoOpen] = useState(false);

    const [postits, setPostits] = useState<PostIt[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        apiInstance
            .get<string[]>('/postit/promotions')
            .then(res => setPromotions(res.data))
            .catch(() => {});
    }, []);

    useFocusEffect(
        useCallback(() => {
            let cancelled = false;
            const fetch = (showLoader: boolean) => {
                if (showLoader) setLoading(true);
                setError(null);
                const params: Record<string, string> = { months: String(months) };
                if (promotion) params.promotion = promotion;
                apiInstance
                    .get<PostItRow[]>('/postit', { params })
                    .then(res => { if (!cancelled) setPostits(res.data.map(toPostIt)); })
                    .catch(err => { if (!cancelled) setError(err.message); })
                    .finally(() => { if (!cancelled) setLoading(false); });
            };

            fetch(true);
            const interval = setInterval(() => fetch(false), 3 * 60_000);
            return () => { cancelled = true; clearInterval(interval); };
        }, [months, promotion])
    );

    const monthOptions = MONTH_OPTIONS.map(m => ({ value: m, label: m === 1 ? '1 mois' : `${m} mois` }));
    const promoOptions = [
        { value: '' as string, label: 'Toutes' },
        ...promotions.map(p => ({ value: p, label: p })),
    ];

    return (
        <View style={styles.container}>
            <ZenBackground />
            <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />

            <View style={styles.filters}>
                <TouchableOpacity
                    style={[styles.filterBtn, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}
                    onPress={() => setMonthsOpen(true)}
                >
                    <Text style={[styles.filterBtnText, { color: colors.textPrimary }]}>
                        {months === 1 ? '1 mois' : `${months} mois`}
                    </Text>
                    <Text style={[styles.filterCaret, { color: colors.textSecondary }]}>▾</Text>
                </TouchableOpacity>

                {promotions.length > 0 && (
                    <TouchableOpacity
                        style={[styles.filterBtn, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}
                        onPress={() => setPromoOpen(true)}
                    >
                        <Text style={[styles.filterBtnText, { color: colors.textPrimary }]}>
                            {promotion || 'Toutes promos'}
                        </Text>
                        <Text style={[styles.filterCaret, { color: colors.textSecondary }]}>▾</Text>
                    </TouchableOpacity>
                )}
            </View>

            <DropdownModal
                visible={monthsOpen}
                onClose={() => setMonthsOpen(false)}
                options={monthOptions}
                selected={months}
                onSelect={setMonths}
                label="Période"
            />
            <DropdownModal
                visible={promoOpen}
                onClose={() => setPromoOpen(false)}
                options={promoOptions}
                selected={promotion}
                onSelect={setPromotion}
                label="Promotion"
            />

            {loading && <ActivityIndicator style={styles.loader} size="large" />}
            {error && <Text style={styles.error}>{error}</Text>}
            {!loading && !error && (
                <FlatList
                    data={postits}
                    keyExtractor={n => String(n.id)}
                    numColumns={2}
                    renderItem={({ item }) => <Note item={item} />}
                    contentContainerStyle={styles.grid}
                    columnWrapperStyle={postits.length > 0 ? styles.row : undefined}
                    style={styles.board}
                />
            )}
        </View>
    );
}

// ─── Styles ───────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: '#c8924a' },

    board: { backgroundColor: 'transparent' },

    // Filters
    filters: { flexDirection: 'row', gap: 8, paddingHorizontal: 12, paddingVertical: 8 },
    filterBtn: { flexDirection: 'row', alignItems: 'center', gap: 4, paddingHorizontal: 12, paddingVertical: 6, borderRadius: 16, borderWidth: 1 },
    filterBtnText: { fontSize: 13, fontWeight: '500' },
    filterCaret: { fontSize: 10 },

    // Dropdown
    overlay: { flex: 1, backgroundColor: 'rgba(0,0,0,0.3)', justifyContent: 'center', alignItems: 'center' },
    dropdown: { backgroundColor: '#fff', borderRadius: 12, paddingVertical: 8, minWidth: 180, boxShadow: '0 4px 8px rgba(0,0,0,0.15)', elevation: 8 },
    dropdownTitle: { fontSize: 11, fontWeight: '700', color: '#999', textTransform: 'uppercase', letterSpacing: 0.5, paddingHorizontal: 16, paddingBottom: 6, borderBottomWidth: 1, borderBottomColor: '#f0f0f0', marginBottom: 4 },
    dropdownItem: { paddingHorizontal: 16, paddingVertical: 10 },
    dropdownItemActive: { backgroundColor: '#e8f0fe' },
    dropdownItemText: { fontSize: 14, color: '#333' },
    dropdownItemTextActive: { color: '#1976d2', fontWeight: '600' },

    // Note card
    grid: { padding: 12, paddingBottom: 32 },
    row: { justifyContent: 'space-between', marginBottom: 20 },
    noteWrapper: { width: '47%' },
    note: {
        borderRadius: 2, padding: 12, paddingTop: 18, minHeight: 160,
        boxShadow: '2px 4px 6px rgba(0,0,0,0.25)', elevation: 6,
    },
    pin: {
        position: 'absolute', top: -7, alignSelf: 'center',
        width: 14, height: 14, borderRadius: 7,
        backgroundColor: '#dc2626',
        boxShadow: '0 2px 2px rgba(0,0,0,0.4)', elevation: 4, zIndex: 10,
    },
    noteAuteur: { fontSize: 9, fontWeight: '700', letterSpacing: 0.8, color: '#6b7280', marginBottom: 6, textTransform: 'uppercase' },
    noteModere: { fontSize: 11, lineHeight: 15, color: '#374151', fontStyle: 'italic', marginBottom: 4 },
    noteReponse: { fontSize: 12, lineHeight: 17, color: '#1f2937', flex: 1 },
    noteFooter: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 8 },
    noteTime: { fontSize: 9, color: '#9ca3af' },
    chip: { backgroundColor: 'rgba(0,0,0,0.08)', borderRadius: 4, paddingHorizontal: 5, paddingVertical: 2 },
    chipText: { fontSize: 9, fontWeight: '700', color: '#374151', textTransform: 'uppercase', letterSpacing: 0.4 },
    urgenceChip: { backgroundColor: '#fee2e2', alignSelf: 'flex-start', marginTop: 5 },
    urgenceText: { fontSize: 9, fontWeight: '700', color: '#dc2626' },

    // Detail modal
    detailBox: {
        width: '85%', maxHeight: '80%', borderRadius: 4, padding: 16, paddingTop: 22,
        boxShadow: '0 8px 12px rgba(0,0,0,0.3)', elevation: 12,
    },
    detailPin: {
        position: 'absolute', top: -9, alignSelf: 'center',
        width: 18, height: 18, borderRadius: 9,
        backgroundColor: '#dc2626',
        boxShadow: '0 2px 2px rgba(0,0,0,0.4)', elevation: 4, zIndex: 10,
    },
    detailAuteur: { fontSize: 10, fontWeight: '700', letterSpacing: 0.8, color: '#6b7280', textTransform: 'uppercase', marginBottom: 10 },
    detailSection: { marginBottom: 10 },
    detailLabel: { fontSize: 10, fontWeight: '700', color: '#6b7280', marginBottom: 3 },
    detailBody: { fontSize: 13, lineHeight: 18, color: '#1f2937' },
    detailItalic: { fontStyle: 'italic', color: '#374151' },
    detailFooter: { flexDirection: 'row', alignItems: 'center', gap: 6, marginTop: 10, flexWrap: 'wrap' },
    detailTime: { fontSize: 10, color: '#9ca3af', marginLeft: 'auto' },
    detailUrgence: { fontSize: 9, fontWeight: '700', color: '#dc2626', backgroundColor: '#fee2e2', borderRadius: 4, paddingHorizontal: 5, paddingVertical: 2 },

    // Misc
    loader: { marginTop: 40 },
    error: { margin: 24, color: '#d32f2f', textAlign: 'center' },
});
