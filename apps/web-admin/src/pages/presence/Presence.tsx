import { useState, useEffect, useCallback } from 'react';
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
import { useAnneePromo } from '../../hooks/useCurriculum';
import { AnneePromoSelect } from '../../components/AnneePromoSelect';
import { ExportSemestreDialog } from './ExportSemestreDialog';
import { ENDPOINT_PRESENCE } from './def';

// ─── Types ────────────────────────────────────────────────────────────────────

interface SeanceCreneau {
    id: number;
    starts_at: string;
    ends_at: string;
    salle: string;
    prof: string;
    promo: string;
}

interface OpenSeanceResponse {
    id: number;
    code: string;
    opened_at: string;
    closed_at: string | null;
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
    hors_groupe?: boolean;
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
    return s.starts_at;
}

function formatSeanceLabel(s: SeanceCreneau): string {
    const start = new Date(s.starts_at);
    const day = start.toLocaleDateString('fr-FR', { weekday: 'short', day: 'numeric', month: 'long' });
    const hd = start.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
    const hf = s.ends_at ? new Date(s.ends_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' }) : '';
    const parts = [day, hf ? `${hd}–${hf}` : hd];
    if (s.salle) parts.push(s.salle);
    if (s.promo) parts.push(s.promo);
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
    const {
        annees, anneesLoading, annee, setSelectedAnnee,
        tree, treeLoading, selectedPromoId, setSelectedPromoId, selectedPromo,
        selectedPeriodeId, setSelectedPeriodeId, selectedPeriode,
        selectedMatiereId, setSelectedMatiereId,
    } = useAnneePromo('presence');

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

    const [selectedSeanceKey, setSelectedSeanceKeyState] = useState<string | null>(
        () => sessionStorage.getItem('presence.seanceKey')
    );
    const setSelectedSeanceKey = useCallback((k: string | null) => {
        setSelectedSeanceKeyState(k);
        if (k == null) sessionStorage.removeItem('presence.seanceKey');
        else sessionStorage.setItem('presence.seanceKey', k);
    }, []);

    // Valide la séance mémorisée contre le planning chargé de la matière courante ;
    // si elle n'y figure plus (changement de matière), on réinitialise. On ignore un
    // planning vide / en cours de chargement pour ne pas perdre le choix au montage.
    useEffect(() => {
        if (!planning || planning.length === 0) return;
        if (selectedSeanceKey != null && !planning.some(s => seanceKey(s) === selectedSeanceKey)) {
            setSelectedSeanceKey(null);
        }
    }, [planning, selectedSeanceKey, setSelectedSeanceKey]);

    const selectedSeance = sortedPlanning.find(s => seanceKey(s) === selectedSeanceKey) ?? null;

    // Séance active
    const [activeSeance, setActiveSeance] = useState<OpenSeanceResponse | null>(null);
    const [seanceClosed, setSeanceClosed] = useState(false);
    const [downloading, setDownloading] = useState(false);
    const [exportOpen, setExportOpen] = useState(false);

    const handleDownloadPdf = useCallback(async () => {
        if (!activeSeance) return;
        setDownloading(true);
        try {
            const resp = await apiInstance.get(
                `${ENDPOINT_PRESENCE}/seance/${activeSeance.id}/pdf`,
                { responseType: 'blob' },
            );
            const url = URL.createObjectURL(new Blob([resp.data], { type: 'application/pdf' }));
            const a = document.createElement('a');
            a.href = url;
            a.download = `presence-seance-${activeSeance.id}.pdf`;
            document.body.appendChild(a);
            a.click();
            a.remove();
            URL.revokeObjectURL(url);
        } finally {
            setDownloading(false);
        }
    }, [activeSeance]);

    // Chargement automatique de l'état du créneau sélectionné
    const { data: slotData } = useQuery<OpenSeanceResponse | null>({
        queryKey: ['seance-slot', selectedMatiereId, selectedSeanceKey],
        queryFn: async () => {
            if (!selectedMatiereId || !selectedSeance) return null;
            try {
                const r = await apiInstance.get<OpenSeanceResponse>(
                    `${ENDPOINT_PRESENCE}/matieres/${selectedMatiereId}/seance/slot?starts_at=${encodeURIComponent(selectedSeance.starts_at)}`
                );
                return r.data;
            } catch (e: any) {
                if (e.response?.status === 404) return null;
                throw e;
            }
        },
        enabled: selectedMatiereId != null && selectedSeance != null,
    });
    useEffect(() => {
        if (slotData === undefined) return;
        setActiveSeance(slotData);
        setSeanceClosed(!!slotData?.closed_at);
    }, [slotData]);

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
        mutationFn: () =>
            apiInstance
                .post<OpenSeanceResponse>(`${ENDPOINT_PRESENCE}/seance`, {
                    matiere_id: selectedMatiereId,
                    starts_at: selectedSeance?.starts_at,
                    ends_at: selectedSeance?.ends_at,
                    salle: selectedSeance?.salle ?? '',
                    prof: selectedSeance?.prof ?? '',
                })
                .then(r => r.data),
        onSuccess: (data) => {
            setActiveSeance(data);
            setSeanceClosed(false);
            queryClient.invalidateQueries({ queryKey: ['presence-live'] });
            queryClient.invalidateQueries({ queryKey: ['seance-slot', selectedMatiereId, selectedSeanceKey] });
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
            queryClient.invalidateQueries({ queryKey: ['seance-slot', selectedMatiereId, selectedSeanceKey] });
        },
    });

    const canOpen = selectedMatiereId != null && selectedSeance != null && !activeSeance;

    return (
        <Box sx={{ p: 3, maxWidth: 960, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={3}>Pointage de présence</Typography>

            {/* Fil d'Ariane */}
            <Paper elevation={0} sx={{ p: 2, mb: 3, border: '1px solid', borderColor: 'divider', display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
                <AnneePromoSelect
                    annees={annees}
                    anneesLoading={anneesLoading}
                    annee={annee}
                    onAnneeChange={a => { setSelectedAnnee(a); setActiveSeance(null); setSeanceClosed(false); }}
                    tree={tree}
                    treeLoading={treeLoading}
                    promoId={selectedPromoId}
                    onPromoChange={id => { setSelectedPromoId(id); setActiveSeance(null); setSeanceClosed(false); }}
                />

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

                {/* L'export porte sur le semestre entier : il n'attend pas
                    qu'une matière, ni une séance, soit sélectionnée. */}
                {selectedPeriodeId != null && (
                    <Button
                        variant="outlined"
                        onClick={() => setExportOpen(true)}
                        sx={{ ml: 'auto', whiteSpace: 'nowrap' }}
                    >
                        Exporter le semestre
                    </Button>
                )}
            </Paper>

            {selectedPeriodeId != null && (
                <ExportSemestreDialog
                    open={exportOpen}
                    periodeId={selectedPeriodeId}
                    onClose={() => setExportOpen(false)}
                />
            )}

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
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
                    <Alert severity="info" sx={{ flex: 1, m: 0 }}>
                        Séance clôturée — le tableau ci-dessous est figé.
                    </Alert>
                    <Button
                        variant="outlined"
                        disabled={downloading}
                        onClick={handleDownloadPdf}
                        sx={{ whiteSpace: 'nowrap' }}
                    >
                        {downloading ? <CircularProgress size={16} sx={{ mr: 1 }} /> : null}
                        Télécharger le PDF
                    </Button>
                </Box>
            )}

            {/* Metrics + tableau */}
            {presenceData && (
                <>
                    <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
                        <MetricCard label="Présents" value={presenceData.presents} color="#16a34a" />
                        <MetricCard label="Retards" value={presenceData.retards} color="#d97706" />
                        <MetricCard label="Absents" value={presenceData.absents} color="#dc2626" />
                        {/* Même convention que le PDF : un retard compte comme
                            une présence, les hors-groupe sont hors du ratio. */}
                        <MetricCard
                            label="Taux de présence"
                            value={presenceData.total > 0
                                ? Math.round(((presenceData.presents + presenceData.retards) / presenceData.total) * 100)
                                : 0}
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
                                                <Avatar sx={{ width: 28, height: 28, fontSize: 11, bgcolor: eleve.hors_groupe ? '#9ca3af' : '#6366f1' }}>
                                                    {eleve.surname[0]}{eleve.name[0]}
                                                </Avatar>
                                                <Typography variant="body2">
                                                    {eleve.surname} {eleve.name}
                                                </Typography>
                                                {eleve.hors_groupe && (
                                                    <Typography variant="caption" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                                                        (H.G.)
                                                    </Typography>
                                                )}
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
