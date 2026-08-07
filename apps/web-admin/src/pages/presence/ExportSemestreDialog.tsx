import { useState, useMemo, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
import List from '@mui/material/List';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { apiInstance } from '../../services/api';
import { ENDPOINT_PRESENCE } from './def';

// ─── Types ────────────────────────────────────────────────────────────────────

interface ExportMatiere {
    id: number;
    name: string;
    nb_seances: number;
    nb_non_cloturees: number;
    premiere: string;
    derniere: string;
}

interface ExportSummary {
    periode_id: number;
    periode: string;
    promo: string;
    annee: number;
    nb_eleves: number;
    du: string;
    au: string;
    matieres: ExportMatiere[];
}

interface DocOptions {
    cover: boolean;
    recap: boolean;
    signature: boolean;
    ledger: boolean;
}

// ─── Estimation ───────────────────────────────────────────────────────────────

// Lignes élèves tenant sur une page de feuille, en-tête et statistiques déduits.
const ELEVES_PAR_PAGE = 28;
// Colonnes séance tenant sur une page de récapitulatif en paysage.
const SEANCES_PAR_PAGE_RECAP = 25;
// Lignes de l'annexe registre tenant sur une page.
const SEANCES_PAR_PAGE_LEDGER = 45;

function estimePages(nbSeances: number, nbEleves: number, opts: DocOptions): number {
    if (nbSeances === 0) return 1;

    const parFeuille = 1 + Math.ceil(nbEleves / ELEVES_PAR_PAGE);
    let total = nbSeances * parFeuille;
    if (opts.cover) total += 2; // page de garde + sommaire
    if (opts.recap) total += Math.ceil(nbSeances / SEANCES_PAR_PAGE_RECAP);
    if (opts.ledger) total += Math.ceil(nbSeances / SEANCES_PAR_PAGE_LEDGER);
    return total;
}

// ─── Composant ────────────────────────────────────────────────────────────────

interface Props {
    open: boolean;
    periodeId: number;
    onClose: () => void;
}

export function ExportSemestreDialog({ open, periodeId, onClose }: Props) {
    const { data: summary, isLoading } = useQuery<ExportSummary>({
        queryKey: ['export-summary', periodeId],
        queryFn: () =>
            apiInstance
                .get<ExportSummary>(`${ENDPOINT_PRESENCE}/periodes/${periodeId}/export-summary`)
                .then(r => r.data),
        enabled: open,
    });

    // null = toutes les matières (état initial, avant tout décochage).
    const [selection, setSelection] = useState<Set<number> | null>(null);
    // null = pas encore saisi : la borne suit alors l'étendue du semestre. Ce
    // repli à la lecture évite de recopier le résumé dans l'état par un effet.
    const [duSaisi, setDuSaisi] = useState<string | null>(null);
    const [auSaisi, setAuSaisi] = useState<string | null>(null);
    const [options, setOptions] = useState<DocOptions>({
        cover: true,
        recap: true,
        signature: false,
        ledger: true,
    });
    const [generating, setGenerating] = useState(false);
    const [erreur, setErreur] = useState<string | null>(null);

    const du = duSaisi ?? summary?.du ?? '';
    const au = auSaisi ?? summary?.au ?? '';

    const matieres = useMemo(() => summary?.matieres ?? [], [summary]);
    const estSelectionnee = useCallback(
        (id: number) => selection === null || selection.has(id),
        [selection],
    );

    const toggleMatiere = useCallback((id: number) => {
        setSelection(prev => {
            const base = prev === null ? new Set(matieres.map(m => m.id)) : new Set(prev);
            if (base.has(id)) base.delete(id);
            else base.add(id);
            return base;
        });
    }, [matieres]);

    const toutes = selection === null || selection.size === matieres.length;
    const toggleToutes = useCallback(() => {
        setSelection(toutes ? new Set<number>() : null);
    }, [toutes]);

    // ── Volumétrie ────────────────────────────────────────────────────────────
    const { nbMatieres, nbSeances, nbExclues, pages } = useMemo(() => {
        const retenues = matieres.filter(m => estSelectionnee(m.id));
        const exclues = retenues.reduce((n, m) => n + m.nb_non_cloturees, 0);
        // Seules les séances clôturées produisent une feuille.
        const seances = retenues.reduce((n, m) => n + m.nb_seances, 0) - exclues;
        return {
            nbMatieres: retenues.length,
            nbSeances: seances,
            nbExclues: exclues,
            pages: estimePages(seances, summary?.nb_eleves ?? 0, options),
        };
    }, [matieres, estSelectionnee, summary, options]);

    const setOption = useCallback((cle: keyof DocOptions) => {
        setOptions(o => ({ ...o, [cle]: !o[cle] }));
    }, []);

    // ── Génération ────────────────────────────────────────────────────────────
    const handleGenerer = useCallback(async () => {
        setGenerating(true);
        setErreur(null);
        try {
            const params = new URLSearchParams();
            if (selection !== null) params.set('matiere_ids', [...selection].join(','));
            if (du) params.set('from', du);
            if (au) params.set('to', au);
            params.set('cover', String(options.cover));
            params.set('recap', String(options.recap));
            params.set('signature', String(options.signature));
            params.set('ledger', String(options.ledger));

            const resp = await apiInstance.get(
                `${ENDPOINT_PRESENCE}/periodes/${periodeId}/pdf?${params.toString()}`,
                { responseType: 'blob' },
            );

            // Le serveur compose un nom lisible ; on le reprend s'il est présent.
            const dispo = resp.headers['content-disposition'] as string | undefined;
            const trouve = dispo?.match(/filename="([^"]+)"/);
            const nom = trouve?.[1] ?? `presence-periode-${periodeId}.pdf`;

            const url = URL.createObjectURL(new Blob([resp.data], { type: 'application/pdf' }));
            const a = document.createElement('a');
            a.href = url;
            a.download = nom;
            document.body.appendChild(a);
            a.click();
            a.remove();
            URL.revokeObjectURL(url);
            onClose();
        } catch {
            setErreur("La génération du document a échoué. Réessayez ou réduisez l'étendue de l'export.");
        } finally {
            setGenerating(false);
        }
    }, [selection, du, au, options, periodeId, onClose]);

    return (
        <Dialog open={open} onClose={generating ? undefined : onClose} maxWidth="md" fullWidth>
            <DialogTitle sx={{ pb: 1 }}>
                Exporter le semestre
                {summary && (
                    <Typography variant="body2" color="text.secondary">
                        {summary.promo} · {summary.periode} · {summary.annee}-{summary.annee + 1} · {summary.nb_eleves} élèves
                    </Typography>
                )}
            </DialogTitle>

            <DialogContent dividers>
                {isLoading && (
                    <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                        <CircularProgress size={28} />
                    </Box>
                )}

                {summary && (
                    <>
                        {/* Matières */}
                        <Typography variant="subtitle2" fontWeight={700} mb={0.5}>Matières</Typography>
                        <Paper variant="outlined" sx={{ maxHeight: 260, overflow: 'auto', mb: 2.5 }}>
                            <List dense disablePadding>
                                <ListItemButton onClick={toggleToutes} sx={{ borderBottom: '1px solid', borderColor: 'divider' }}>
                                    <Checkbox
                                        edge="start"
                                        checked={toutes}
                                        indeterminate={!toutes && (selection?.size ?? 0) > 0}
                                        tabIndex={-1}
                                        disableRipple
                                    />
                                    <ListItemText
                                        primary="Toutes les matières"
                                        primaryTypographyProps={{ fontWeight: 600 }}
                                    />
                                </ListItemButton>

                                {matieres.map(m => (
                                    <ListItemButton key={m.id} onClick={() => toggleMatiere(m.id)}>
                                        <Checkbox edge="start" checked={estSelectionnee(m.id)} tabIndex={-1} disableRipple />
                                        <ListItemText
                                            primary={m.name}
                                            secondary={m.premiere ? `${m.premiere} → ${m.derniere}` : 'aucune séance planifiée'}
                                        />
                                        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', ml: 2 }}>
                                            {m.nb_non_cloturees > 0 && (
                                                <Chip
                                                    size="small"
                                                    color="warning"
                                                    variant="outlined"
                                                    label={`${m.nb_non_cloturees} non clôturée${m.nb_non_cloturees > 1 ? 's' : ''}`}
                                                />
                                            )}
                                            <Chip
                                                size="small"
                                                variant="outlined"
                                                label={`${m.nb_seances} séance${m.nb_seances > 1 ? 's' : ''}`}
                                            />
                                        </Box>
                                    </ListItemButton>
                                ))}
                            </List>
                        </Paper>

                        {/* Dates */}
                        <Typography variant="subtitle2" fontWeight={700} mb={1}>Période</Typography>
                        <Box sx={{ display: 'flex', gap: 2, mb: 2.5, flexWrap: 'wrap' }}>
                            <TextField
                                label="Du" type="date" size="small" value={du}
                                onChange={e => setDuSaisi(e.target.value)}
                                InputLabelProps={{ shrink: true }}
                            />
                            <TextField
                                label="Au" type="date" size="small" value={au}
                                onChange={e => setAuSaisi(e.target.value)}
                                InputLabelProps={{ shrink: true }}
                            />
                        </Box>

                        {/* Options du document */}
                        <Typography variant="subtitle2" fontWeight={700} mb={0.5}>Document</Typography>
                        <Box sx={{ display: 'flex', flexDirection: 'column', mb: 1 }}>
                            <FormControlLabel
                                control={<Checkbox size="small" checked={options.cover} onChange={() => setOption('cover')} />}
                                label="Page de garde et sommaire"
                            />
                            <FormControlLabel
                                control={<Checkbox size="small" checked={options.recap} onChange={() => setOption('recap')} />}
                                label="Récapitulatif par élève"
                            />
                            <FormControlLabel
                                control={<Checkbox size="small" checked={options.signature} onChange={() => setOption('signature')} />}
                                label="Colonne signature vierge"
                            />
                            <FormControlLabel
                                control={<Checkbox size="small" checked={options.ledger} onChange={() => setOption('ledger')} />}
                                label="Annexe registre d'intégrité"
                            />
                        </Box>

                        {nbExclues > 0 && (
                            <Alert severity="warning" sx={{ mt: 1 }}>
                                {nbExclues} séance{nbExclues > 1 ? 's' : ''} non clôturée{nbExclues > 1 ? 's' : ''} :
                                son pointage étant encore ouvert, elle{nbExclues > 1 ? 's' : ''} n'apparaîtra
                                {nbExclues > 1 ? 'ont' : ''} pas dans le corps du document mais {nbExclues > 1 ? 'seront listées' : 'sera listée'} nommément
                                en page de garde. Clôturez-la{nbExclues > 1 ? 'es' : ''} puis relancez l'export pour l'y intégrer.
                            </Alert>
                        )}

                        {erreur && <Alert severity="error" sx={{ mt: 2 }}>{erreur}</Alert>}
                    </>
                )}
            </DialogContent>

            <Divider />
            <DialogActions sx={{ px: 3, py: 1.5, justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">
                    {nbMatieres} matière{nbMatieres > 1 ? 's' : ''} · {nbSeances} séance{nbSeances > 1 ? 's' : ''} · environ {pages} page{pages > 1 ? 's' : ''}
                </Typography>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button onClick={onClose} disabled={generating}>Annuler</Button>
                    <Button
                        variant="contained"
                        onClick={handleGenerer}
                        disabled={generating || nbSeances === 0}
                        sx={{ bgcolor: '#4f46e5', '&:hover': { bgcolor: '#4338ca' } }}
                    >
                        {generating ? <CircularProgress size={16} sx={{ mr: 1, color: '#fff' }} /> : null}
                        Générer le PDF
                    </Button>
                </Box>
            </DialogActions>
        </Dialog>
    );
}
