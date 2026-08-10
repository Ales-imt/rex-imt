import { Fragment, useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Collapse from '@mui/material/Collapse';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Select from '@mui/material/Select';
import Switch from '@mui/material/Switch';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { apiInstance } from '../../services/api';
import { useAnneePromo } from '../../hooks/useCurriculum';
import { AnneePromoSelect } from '../../components/AnneePromoSelect';
import { PRESENCE_WORKFLOW } from '../presence/def';
import { JustificationDialog } from './JustificationDialog';
import { ENDPOINT_JUSTIFICATION } from './def';
import { formatPlageFR, formatSeanceFR, nomSaisissant, STATUT_JUSTIFICATION } from './format';
import type { Justification, SeanceApercu } from './types';

// Page de gestion des excuses. Vocabulaire : on « annule » une excuse, on ne la
// « supprime » jamais — rien n'est supprimé en base, et les libellés doivent
// refléter ce que le système fait réellement.

function LigneSeances({ id }: { id: number }) {
    const { data, isLoading } = useQuery<SeanceApercu[]>({
        queryKey: ['justification-seances', id],
        queryFn: () => apiInstance
            .get<SeanceApercu[]>(`${ENDPOINT_JUSTIFICATION}/${id}/seances`)
            .then(r => r.data),
    });

    if (isLoading) return <CircularProgress size={18} sx={{ m: 2 }} />;
    if (!data || data.length === 0) {
        return (
            <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                Aucune séance planifiée sur cette période.
            </Typography>
        );
    }
    return (
        <Box sx={{ py: 1, pl: 4 }}>
            {data.map(s => (
                <Typography key={s.id} variant="body2" color="text.secondary">
                    {s.matiere} · {formatSeanceFR(s)}
                </Typography>
            ))}
        </Box>
    );
}

export function Justifications() {
    const theme = useTheme();
    const queryClient = useQueryClient();
    const navigate = useNavigate();

    const {
        annees, anneesLoading, annee, setSelectedAnnee,
        tree, treeLoading, selectedPromoId, setSelectedPromoId, selectedPromo,
    } = useAnneePromo('justification');

    const [recherche, setRecherche] = useState('');
    const [avecHistorique, setAvecHistorique] = useState(false);
    const [filtreSemestre, setFiltreSemestre] = useState<number | ''>('');
    const [deplie, setDeplie] = useState<number | null>(null);
    const [enEdition, setEnEdition] = useState<Justification | null>(null);
    const [aAnnuler, setAAnnuler] = useState<Justification | null>(null);

    const params = useMemo(() => {
        const p = new URLSearchParams();
        if (selectedPromoId != null) p.set('promo_id', String(selectedPromoId));
        if (filtreSemestre !== '') p.set('periode_id', String(filtreSemestre));
        if (recherche.trim() !== '') p.set('q', recherche.trim());
        if (avecHistorique) p.set('include_revoked', 'true');
        return p;
    }, [selectedPromoId, filtreSemestre, recherche, avecHistorique]);

    const { data: justifications, isLoading } = useQuery<Justification[]>({
        queryKey: ['justifications', params.toString()],
        queryFn: () => apiInstance
            .get<Justification[]>(`${ENDPOINT_JUSTIFICATION}?${params}`)
            .then(r => r.data),
        enabled: selectedPromoId != null,
    });

    const annuler = useMutation({
        mutationFn: (id: number) => apiInstance.delete(`${ENDPOINT_JUSTIFICATION}/${id}`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['justifications'] });
            queryClient.invalidateQueries({ queryKey: ['presence-live'] });
            setAAnnuler(null);
        },
    });

    // Une chaîne de corrections doit rester parcourable dans les deux sens :
    // depuis la version remplacée vers celle qui la remplace, et retour.
    const parID = useMemo(
        () => new Map((justifications ?? []).map(j => [j.id, j])),
        [justifications],
    );
    // Atteindre l'autre version : elle peut être hors de la liste courante
    // (l'historique est masqué par défaut), auquel cas on l'affiche.
    const versVersion = (id: number, libelle: string) => (
        <Link
            component="button"
            type="button"
            underline="hover"
            onClick={() => {
                if (!parID.has(id)) setAvecHistorique(true);
                setDeplie(id);
            }}
            sx={{ fontSize: 'inherit' }}
        >
            {libelle}
        </Link>
    );

    return (
        <Box sx={{ p: 3, maxWidth: 1100, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={3}>Excuses d'absence</Typography>

            <Paper elevation={0} sx={{ p: 2, mb: 3, border: '1px solid', borderColor: 'divider', display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
                <AnneePromoSelect
                    annees={annees}
                    anneesLoading={anneesLoading}
                    annee={annee}
                    onAnneeChange={setSelectedAnnee}
                    tree={tree}
                    treeLoading={treeLoading}
                    promoId={selectedPromoId}
                    onPromoChange={setSelectedPromoId}
                />

                {selectedPromo && (
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Semestre</InputLabel>
                        <Select
                            value={filtreSemestre}
                            label="Semestre"
                            onChange={e => {
                                const v = e.target.value;
                                setFiltreSemestre(typeof v === 'number' ? v : '');
                            }}
                        >
                            <MenuItem value="">Tous</MenuItem>
                            {selectedPromo.periodes.map(p => (
                                <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                )}

                <TextField
                    size="small"
                    label="Étudiant"
                    value={recherche}
                    onChange={e => setRecherche(e.target.value)}
                    sx={{ minWidth: 200 }}
                />

                <FormControlLabel
                    sx={{ ml: 'auto' }}
                    control={
                        <Switch
                            size="small"
                            checked={avecHistorique}
                            onChange={e => setAvecHistorique(e.target.checked)}
                        />
                    }
                    label="Afficher l'historique"
                />
            </Paper>

            {isLoading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                    <CircularProgress size={28} />
                </Box>
            )}

            {justifications && justifications.length === 0 && (
                <Paper elevation={0} sx={{ p: 4, border: '1px solid', borderColor: 'divider', textAlign: 'center' }}>
                    <Typography variant="body1" color="text.secondary" mb={2}>
                        Aucune excuse enregistrée pour cette promotion
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                        Une excuse se saisit depuis la feuille de présence, sur la ligne de l'étudiant
                        concerné : le créneau de la séance y pré-remplit la plage.
                    </Typography>
                    {/* navigate, et non href : un lien natif rechargerait tout
                        le document — flash blanc et réamorçage de l'application. */}
                    <Button variant="outlined" onClick={() => navigate(`/${PRESENCE_WORKFLOW}`)} sx={{ mt: 2 }}>
                        Aller au pointage de présence
                    </Button>
                </Paper>
            )}

            {justifications && justifications.length > 0 && (
                <TableContainer component={Paper} elevation={0} sx={{ border: '1px solid', borderColor: 'divider' }}>
                    <Table size="small">
                        <TableHead>
                            <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? '#1e293b' : '#f8fafc' }}>
                                <TableCell>Étudiant</TableCell>
                                <TableCell>Période</TableCell>
                                <TableCell align="right">Séances</TableCell>
                                <TableCell>Saisie par</TableCell>
                                <TableCell>Statut</TableCell>
                                <TableCell align="right">Actions</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {justifications.map(j => {
                                const statut = STATUT_JUSTIFICATION[j.statut];
                                return (
                                  <Fragment key={j.id}>
                                    <TableRow
                                        hover
                                        selected={deplie === j.id}
                                    >
                                        <TableCell>
                                            <Typography variant="body2">{j.surname} {j.name}</Typography>
                                        </TableCell>
                                        <TableCell>
                                            <Typography variant="body2">
                                                {formatPlageFR(j.starts_at, j.ends_at)}
                                            </Typography>
                                            {(j.replaces_id != null || j.replaced_by_id != null) && (
                                                <Typography variant="caption" color="text.secondary" component="div">
                                                    {j.replaced_by_id != null && versVersion(j.replaced_by_id, 'remplacée par une version plus récente')}
                                                    {j.replaces_id != null && j.replaced_by_id != null && ' · '}
                                                    {j.replaces_id != null && versVersion(j.replaces_id, 'corrige une version précédente')}
                                                </Typography>
                                            )}
                                        </TableCell>
                                        <TableCell align="right">
                                            <Link
                                                component="button"
                                                type="button"
                                                underline="hover"
                                                onClick={() => setDeplie(deplie === j.id ? null : j.id)}
                                            >
                                                {j.nb_seances}
                                            </Link>
                                        </TableCell>
                                        <TableCell>
                                            <Typography variant="body2" color="text.secondary">
                                                {nomSaisissant(j.created_by_surname, j.created_by_name)}
                                            </Typography>
                                        </TableCell>
                                        <TableCell>
                                            <Chip size="small" variant="outlined" label={statut.label} color={statut.color} />
                                        </TableCell>
                                        <TableCell align="right">
                                            {j.statut === 'ACTIVE' && (
                                                <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                                                    <Button size="small" onClick={() => setEnEdition(j)}>Modifier</Button>
                                                    <Button size="small" color="error" onClick={() => setAAnnuler(j)}>
                                                        Annuler l'excuse
                                                    </Button>
                                                </Box>
                                            )}
                                        </TableCell>
                                    </TableRow>
                                    <TableRow>
                                        <TableCell colSpan={6} sx={{ py: 0, borderBottom: deplie === j.id ? undefined : 'none' }}>
                                            <Collapse in={deplie === j.id} unmountOnExit>
                                                <LigneSeances id={j.id} />
                                            </Collapse>
                                        </TableCell>
                                    </TableRow>
                                  </Fragment>
                                );
                            })}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}

            {enEdition && (
                <JustificationDialog
                    userId={enEdition.user_id}
                    eleve={`${enEdition.surname} ${enEdition.name}`}
                    justification={enEdition}
                    onClose={() => setEnEdition(null)}
                />
            )}

            <Dialog open={aAnnuler != null} onClose={() => setAAnnuler(null)}>
                <DialogTitle>Annuler l'excuse</DialogTitle>
                <DialogContent>
                    <DialogContentText>
                        {aAnnuler && (
                            <>
                                {aAnnuler.nb_seances} séance{aAnnuler.nb_seances > 1 ? 's' : ''} redeviendr
                                {aAnnuler.nb_seances > 1 ? 'ont' : 'a'} « Absent » pour {aAnnuler.surname} {aAnnuler.name}.
                                L'excuse reste consultable dans l'historique : rien n'est supprimé.
                            </>
                        )}
                    </DialogContentText>
                    {annuler.isError && (
                        <Alert severity="error" sx={{ mt: 2 }}>
                            L'annulation a échoué. L'excuse a peut-être déjà été annulée par ailleurs.
                        </Alert>
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setAAnnuler(null)} disabled={annuler.isPending}>Retour</Button>
                    <Button
                        color="error"
                        variant="contained"
                        disabled={annuler.isPending}
                        onClick={() => aAnnuler && annuler.mutate(aAnnuler.id)}
                    >
                        {annuler.isPending ? <CircularProgress size={16} sx={{ mr: 1 }} /> : null}
                        Annuler l'excuse
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
