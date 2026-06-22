import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Avatar from '@mui/material/Avatar';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Alert from '@mui/material/Alert';
import { useTheme } from '@mui/material/styles';
import { QRCodeSVG } from 'qrcode.react';
import { apiInstance } from '../../services/api';
import { useAnnees, usePromotionTree } from '../evaluation/useEvaluations';
import { ENDPOINT_PRESENCE } from './def';
import type { PromotionTree } from '../evaluation/Evaluation';

// ─── Types ────────────────────────────────────────────────────────────────────

interface SeanceCreneau {
    date: string;
    hd: string;
    hf: string;
    cours: string;
    salle: string;
    prof: string;
    promo: string;
}

interface OpenSeanceResponse {
    id: number;
    code: string;
    opened_at: string;
    late_after_minutes: number;
}

interface TokenResponse {
    token: string;
    code: string;
    ttl_seconds: number;
}

interface ElevePresence {
    user_id: number;
    name: string;
    surname: string;
    statut: 'PRESENT' | 'RETARD' | 'ABSENT';
    pointe_at: string | null;
}

interface PresenceResponse {
    matiere: string;
    total: number;
    presents: number;
    retards: number;
    absents: number;
    eleves: ElevePresence[];
}

// ─── Statut chip ──────────────────────────────────────────────────────────────

const STATUT_COLOR: Record<string, 'success' | 'warning' | 'error'> = {
    PRESENT: 'success',
    RETARD: 'warning',
    ABSENT: 'error',
};
const STATUT_LABEL: Record<string, string> = {
    PRESENT: 'Présent',
    RETARD: 'Retard',
    ABSENT: 'Absent',
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

function seanceKey(s: SeanceCreneau): string {
    return `${s.date}T${s.hd}`;
}

function formatSeanceLabel(s: SeanceCreneau): string {
    const date = new Date(`${s.date}T12:00:00`);
    const day = date.toLocaleDateString('fr-FR', { weekday: 'short', day: 'numeric', month: 'long' });
    const parts = [day, `${s.hd}–${s.hf}`];
    if (s.salle) parts.push(s.salle);
    return parts.join(' · ');
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function MetricCard({ label, value, color }: { label: string; value: number; color: string }) {
    return (
        <Card elevation={0} sx={{ flex: 1, border: '1px solid', borderColor: 'divider', minWidth: 100 }}>
            <CardContent sx={{ py: 1.5, px: 2, '&:last-child': { pb: 1.5 } }}>
                <Typography variant="h4" fontWeight={700} sx={{ color }}>{value}</Typography>
                <Typography variant="caption" color="text.secondary">{label}</Typography>
            </CardContent>
        </Card>
    );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function Presence() {
    const theme = useTheme();
    const queryClient = useQueryClient();

    // Fil d'Ariane
    const { data: annees, loading: anneesLoading } = useAnnees();
    const [selectedAnnee, setSelectedAnnee] = useState<number | null>(null);
    const annee = selectedAnnee ?? (annees?.[0] ?? null);

    const { data: tree, loading: treeLoading } = usePromotionTree(annee);

    const [selectedPromoId, setSelectedPromoId] = useState<number | null>(null);
    useEffect(() => { setSelectedPromoId(tree?.[0]?.id ?? null); }, [tree]);

    const selectedPromo: PromotionTree | undefined = tree?.find(p => p.id === selectedPromoId);

    const [selectedPeriodeId, setSelectedPeriodeId] = useState<number | null>(null);
    useEffect(() => {
        setSelectedPeriodeId(selectedPromo?.periodes?.[0]?.id ?? null);
    }, [selectedPromo]);

    const selectedPeriode = selectedPromo?.periodes?.find(p => p.id === selectedPeriodeId);

    const [selectedMatiereId, setSelectedMatiereId] = useState<number | null>(null);
    useEffect(() => { setSelectedMatiereId(null); }, [selectedPeriodeId]);

    // Planning de la matière sélectionnée
    const { data: planning, isLoading: planningLoading } = useQuery<SeanceCreneau[]>({
        queryKey: ['planning', selectedMatiereId],
        queryFn: () =>
            apiInstance
                .get<SeanceCreneau[]>(`${ENDPOINT_PRESENCE}/matieres/${selectedMatiereId}/planning`)
                .then(r => r.data),
        enabled: selectedMatiereId != null,
    });

    const sortedPlanning = (planning ?? [])
        .slice()
        .sort((a, b) => seanceKey(a).localeCompare(seanceKey(b)));

    const [selectedSeanceKey, setSelectedSeanceKey] = useState<string | null>(null);
    useEffect(() => { setSelectedSeanceKey(null); }, [selectedMatiereId]);

    const selectedSeance = sortedPlanning.find(s => seanceKey(s) === selectedSeanceKey) ?? null;

    // Séance active
    const [activeSeance, setActiveSeance] = useState<OpenSeanceResponse | null>(null);
    const [seanceClosed, setSeanceClosed] = useState(false);

    // Token QR (rafraîchi ~25s)
    const { data: tokenData } = useQuery<TokenResponse>({
        queryKey: ['presence-token', activeSeance?.id],
        queryFn: () =>
            apiInstance
                .get<TokenResponse>(`${ENDPOINT_PRESENCE}/seance/${activeSeance!.id}/token`)
                .then(r => r.data),
        enabled: activeSeance != null && !seanceClosed,
        refetchInterval: 25_000,
    });

    // Présence en direct (rafraîchi ~5s)
    const { data: presenceData } = useQuery<PresenceResponse>({
        queryKey: ['presence-live', activeSeance?.id],
        queryFn: () =>
            apiInstance
                .get<PresenceResponse>(`${ENDPOINT_PRESENCE}/seance/${activeSeance!.id}/presence`)
                .then(r => r.data),
        enabled: activeSeance != null,
        refetchInterval: seanceClosed ? false : 5_000,
    });

    // Ouvrir une séance
    const openMutation = useMutation({
        mutationFn: () => {
            const starts_at = selectedSeance
                ? new Date(`${selectedSeance.date}T${selectedSeance.hd}:00`).toISOString()
                : undefined;
            const ends_at = selectedSeance
                ? new Date(`${selectedSeance.date}T${selectedSeance.hf}:00`).toISOString()
                : undefined;
            return apiInstance
                .post<OpenSeanceResponse>(`${ENDPOINT_PRESENCE}/seance`, {
                    matiere_id: selectedMatiereId,
                    starts_at,
                    ends_at,
                    salle: selectedSeance?.salle ?? '',
                    prof: selectedSeance?.prof ?? '',
                })
                .then(r => r.data);
        },
        onSuccess: (data) => {
            setActiveSeance(data);
            setSeanceClosed(false);
            queryClient.invalidateQueries({ queryKey: ['presence-live'] });
        },
    });

    // Fermer la séance
    const closeMutation = useMutation({
        mutationFn: () =>
            apiInstance
                .post(`${ENDPOINT_PRESENCE}/seance/${activeSeance!.id}/close`)
                .then(r => r.data),
        onSuccess: () => {
            setSeanceClosed(true);
            queryClient.invalidateQueries({ queryKey: ['presence-live'] });
        },
    });

    const canOpen = selectedMatiereId != null && selectedSeance != null && !activeSeance;

    return (
        <Box sx={{ p: 3, maxWidth: 960, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={3}>Pointage de présence</Typography>

            {/* Fil d'Ariane */}
            <Paper elevation={0} sx={{ p: 2, mb: 3, border: '1px solid', borderColor: 'divider', display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
                {anneesLoading ? <CircularProgress size={18} /> : (
                    <FormControl size="small" sx={{ minWidth: 110 }}>
                        <InputLabel>Année</InputLabel>
                        <Select value={annee ?? ''} label="Année" onChange={e => { setSelectedAnnee(Number(e.target.value)); setActiveSeance(null); setSeanceClosed(false); }}>
                            {(annees ?? []).map(a => <MenuItem key={a} value={a}>{a}/{a + 1}</MenuItem>)}
                        </Select>
                    </FormControl>
                )}

                {!treeLoading && (tree ?? []).length > 0 && (
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Promotion</InputLabel>
                        <Select value={selectedPromoId ?? ''} label="Promotion" onChange={e => { setSelectedPromoId(Number(e.target.value)); setActiveSeance(null); setSeanceClosed(false); }}>
                            {(tree ?? []).map(p => <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>)}
                        </Select>
                    </FormControl>
                )}

                {selectedPromo && (
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Semestre</InputLabel>
                        <Select value={selectedPeriodeId ?? ''} label="Semestre" onChange={e => { setSelectedPeriodeId(Number(e.target.value)); setActiveSeance(null); setSeanceClosed(false); }}>
                            {selectedPromo.periodes.map(p => <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>)}
                        </Select>
                    </FormControl>
                )}

                {selectedPeriode && (
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>Matière</InputLabel>
                        <Select
                            value={selectedMatiereId ?? ''}
                            label="Matière"
                            onChange={e => { setSelectedMatiereId(Number(e.target.value)); setActiveSeance(null); setSeanceClosed(false); }}
                            sx={{ '& .MuiSelect-select': { color: selectedMatiereId ? '#4f46e5' : undefined, fontWeight: selectedMatiereId ? 600 : undefined } }}
                        >
                            {selectedPeriode.matieres.map(m => <MenuItem key={m.id} value={m.id}>{m.name}</MenuItem>)}
                        </Select>
                    </FormControl>
                )}

                {selectedMatiereId != null && (
                    <FormControl size="small" sx={{ minWidth: 240 }}>
                        <InputLabel>Séance</InputLabel>
                        {planningLoading ? (
                            <CircularProgress size={18} />
                        ) : (
                            <Select
                                value={selectedSeanceKey ?? ''}
                                label="Séance"
                                onChange={e => { setSelectedSeanceKey(e.target.value as string); setActiveSeance(null); setSeanceClosed(false); }}
                                sx={{ '& .MuiSelect-select': { color: selectedSeanceKey ? '#4f46e5' : undefined, fontWeight: selectedSeanceKey ? 600 : undefined } }}
                            >
                                {sortedPlanning.map(s => (
                                    <MenuItem key={seanceKey(s)} value={seanceKey(s)}>
                                        {formatSeanceLabel(s)}
                                    </MenuItem>
                                ))}
                            </Select>
                        )}
                    </FormControl>
                )}
            </Paper>

            {/* Bouton ouvrir */}
            {!activeSeance && (
                <Button
                    variant="contained"
                    disabled={!canOpen || openMutation.isPending}
                    onClick={() => openMutation.mutate()}
                    sx={{ mb: 3, bgcolor: '#4f46e5', '&:hover': { bgcolor: '#4338ca' } }}
                >
                    {openMutation.isPending ? <CircularProgress size={18} sx={{ mr: 1, color: '#fff' }} /> : null}
                    Ouvrir le pointage
                </Button>
            )}

            {openMutation.isError && (
                <Alert severity="error" sx={{ mb: 2 }}>Erreur lors de l'ouverture de la séance.</Alert>
            )}

            {/* Bloc QR + code court */}
            {activeSeance && !seanceClosed && tokenData && (
                <Paper elevation={0} sx={{ p: 3, mb: 3, border: '1px solid', borderColor: 'divider', display: 'flex', gap: 4, alignItems: 'center', flexWrap: 'wrap' }}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}>
                        <QRCodeSVG value={tokenData.token} size={260} level="M" bgColor="#ffffff" fgColor="#000000" marginSize={4} />
                        <Typography variant="caption" color="text.secondary">
                            Valide {tokenData.ttl_seconds}s · se rafraîchit automatiquement
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="body2" color="text.secondary" mb={0.5}>Code de repli</Typography>
                        <Typography
                            variant="h3"
                            fontWeight={800}
                            letterSpacing={6}
                            sx={{ fontFamily: 'monospace', color: '#4f46e5' }}
                        >
                            {tokenData.code}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                            Ouverture : {new Date(activeSeance.opened_at).toLocaleTimeString('fr-FR')}
                        </Typography>
                    </Box>
                    <Box sx={{ ml: 'auto' }}>
                        <Button
                            variant="outlined"
                            color="error"
                            disabled={closeMutation.isPending}
                            onClick={() => closeMutation.mutate()}
                        >
                            {closeMutation.isPending ? <CircularProgress size={16} sx={{ mr: 1 }} /> : null}
                            Clôturer la séance
                        </Button>
                    </Box>
                </Paper>
            )}

            {seanceClosed && (
                <Alert severity="info" sx={{ mb: 3 }}>Séance clôturée — le tableau ci-dessous est figé.</Alert>
            )}

            {/* Metrics + tableau */}
            {presenceData && (
                <>
                    <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
                        <MetricCard label="Présents" value={presenceData.presents} color="#16a34a" />
                        <MetricCard label="Retards" value={presenceData.retards} color="#d97706" />
                        <MetricCard label="Absents" value={presenceData.absents} color="#dc2626" />
                        <MetricCard
                            label="Taux de présence"
                            value={presenceData.total > 0 ? Math.round((presenceData.presents / presenceData.total) * 100) : 0}
                            color={theme.palette.text.primary}
                        />
                    </Box>

                    <TableContainer component={Paper} elevation={0} sx={{ border: '1px solid', borderColor: 'divider' }}>
                        <Table size="small">
                            <TableHead>
                                <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? '#1e293b' : '#f8fafc' }}>
                                    <TableCell>Élève</TableCell>
                                    <TableCell>Statut</TableCell>
                                    <TableCell>Pointé à</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {presenceData.eleves.map(eleve => (
                                    <TableRow key={eleve.user_id} hover>
                                        <TableCell>
                                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                                <Avatar sx={{ width: 28, height: 28, fontSize: 11, bgcolor: '#6366f1' }}>
                                                    {eleve.surname[0]}{eleve.name[0]}
                                                </Avatar>
                                                <Typography variant="body2">
                                                    {eleve.surname} {eleve.name}
                                                </Typography>
                                            </Box>
                                        </TableCell>
                                        <TableCell>
                                            <Chip
                                                label={STATUT_LABEL[eleve.statut] ?? eleve.statut}
                                                color={STATUT_COLOR[eleve.statut] ?? 'default'}
                                                size="small"
                                                variant="outlined"
                                            />
                                        </TableCell>
                                        <TableCell>
                                            <Typography variant="body2" color="text.secondary">
                                                {eleve.pointe_at
                                                    ? new Date(eleve.pointe_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
                                                    : '—'}
                                            </Typography>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </>
            )}
        </Box>
    );
}
