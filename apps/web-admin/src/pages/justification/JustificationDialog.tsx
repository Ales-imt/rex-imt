import { useState, useMemo, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
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
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { apiInstance } from '../../services/api';
import { ENDPOINT_JUSTIFICATION } from './def';
import {
    dureeHeures, dureeJours, formatPlageFR, formatSeanceFR,
    fromInputLocal, toInputLocal,
} from './format';
import type { ApercuJustification, Justification } from './types';

// Dialogue unique pour la création et la modification. « Modifier » n'est pas
// un UPDATE côté serveur : l'ancienne excuse est révoquée et une nouvelle
// version la remplace — d'où le PUT, qui retourne la NOUVELLE justification.

// Au-delà de ce seuil, l'enregistrement demande une confirmation explicite.
// Couvre une longue maladie sans laisser passer une année tapée de travers ;
// le refus sec est posé plus haut, à 400 jours, côté serveur.
const SEUIL_CONFIRMATION_JOURS = 30;

const STATUT_POINTAGE: Record<string, { label: string; color: 'success' | 'warning' | 'error' }> = {
    PRESENT: { label: 'Présent', color: 'success' },
    RETARD: { label: 'Retard', color: 'warning' },
    ABSENT: { label: 'Absent', color: 'error' },
};

// messageErreur extrait le message du serveur quand il y en a un : un 409 dit
// « cette excuse vient d'être modifiée par ailleurs », ce qui est infiniment
// plus utile qu'un message générique.
function messageErreur(e: unknown): string {
    const reponse = (e as { response?: { data?: { message?: unknown } } })?.response;
    const message = reponse?.data?.message;
    if (typeof message === 'string' && message !== '') return message;
    return "L'enregistrement a échoué. Rechargez la page : l'excuse a peut-être été modifiée par ailleurs.";
}

// Le dialogue lit ses valeurs initiales à la construction : les appelants le
// MONTENT à l'ouverture ({cible && <JustificationDialog … />}) au lieu de le
// laisser monté en permanence. Pas d'effet de réinitialisation à maintenir, et
// chaque ouverture repart d'un état propre.
interface Props {
    userId: number;
    /** Libellé de l'étudiant, affiché en sous-titre. */
    eleve?: string;
    /** Renseigné en modification ; absent en création. */
    justification?: Justification | null;
    /** Créneau de pré-remplissage en création (la séance d'où l'on vient). */
    creneau?: { starts_at: string; ends_at: string } | null;
    onClose: () => void;
    onSaved?: (j: Justification) => void;
    /** Ouvre une autre excuse depuis l'alerte de chevauchement. */
    onOuvrirAutre?: (id: number) => void;
}

export function JustificationDialog({
    userId, eleve, justification, creneau, onClose, onSaved, onOuvrirAutre,
}: Props) {
    const queryClient = useQueryClient();
    const enModification = justification != null;

    // Pré-remplissage : la plage existante en modification, le créneau de la
    // séance d'origine en création.
    const source = justification
        ? { starts_at: justification.starts_at, ends_at: justification.ends_at }
        : creneau;
    const [debut, setDebut] = useState(() => (source ? toInputLocal(source.starts_at) : ''));
    const [fin, setFin] = useState(() => (source ? toInputLocal(source.ends_at) : ''));
    // Séances explicitement décochées par le gestionnaire. On mémorise les
    // exclusions plutôt que les inclusions : la liste des séances change à
    // chaque modification des bornes, une sélection positive serait à
    // reconstruire en permanence.
    const [exclues, setExclues] = useState<Set<number>>(new Set());
    const [confirmee, setConfirmee] = useState(false);
    const [erreur, setErreur] = useState<string | null>(null);

    const debutISO = fromInputLocal(debut);
    const finISO = fromInputLocal(fin);
    const plageValide = debutISO != null && finISO != null && finISO > debutISO;

    const { data: apercu, isFetching } = useQuery<ApercuJustification>({
        queryKey: ['justification-preview', userId, debutISO, finISO, justification?.id ?? null],
        queryFn: () => {
            const params = new URLSearchParams({
                user_id: String(userId),
                starts_at: debutISO!,
                ends_at: finISO!,
            });
            if (justification) params.set('exclude_id', String(justification.id));
            return apiInstance
                .get<ApercuJustification>(`${ENDPOINT_JUSTIFICATION}/preview?${params}`)
                .then(r => r.data);
        },
        enabled: plageValide,
    });

    const seances = useMemo(() => apercu?.seances ?? [], [apercu]);
    const retenues = useMemo(
        () => seances.filter(s => s.dans_plage && !exclues.has(s.id)),
        [seances, exclues],
    );

    const toggle = useCallback((id: number) => {
        setExclues(prev => {
            const suivant = new Set(prev);
            if (suivant.has(id)) suivant.delete(id);
            else suivant.add(id);
            return suivant;
        });
    }, []);

    const nbJours = plageValide ? dureeJours(debutISO!, finISO!) : 0;
    const confirmationRequise = nbJours > SEUIL_CONFIRMATION_JOURS && !confirmee;

    const enregistrer = useMutation({
        mutationFn: () => {
            const corps = {
                user_id: userId,
                starts_at: debutISO,
                ends_at: finISO,
                seance_ids: retenues.map(s => s.id),
            };
            const requete = enModification
                ? apiInstance.put<Justification>(`${ENDPOINT_JUSTIFICATION}/${justification!.id}`, corps)
                : apiInstance.post<Justification>(ENDPOINT_JUSTIFICATION, corps);
            return requete.then(r => r.data);
        },
        onSuccess: (creee) => {
            queryClient.invalidateQueries({ queryKey: ['justifications'] });
            queryClient.invalidateQueries({ queryKey: ['presence-live'] });
            onSaved?.(creee);
            onClose();
        },
        onError: (e: unknown) => {
            setErreur(messageErreur(e));
        },
    });

    const valider = useCallback(() => {
        setErreur(null);
        if (confirmationRequise) {
            setConfirmee(true);
            return;
        }
        enregistrer.mutate();
    }, [confirmationRequise, enregistrer]);

    const enCours = enregistrer.isPending;

    return (
        <Dialog open onClose={enCours ? undefined : onClose} maxWidth="md" fullWidth>
            <DialogTitle sx={{ pb: 1 }}>
                {enModification ? "Modifier l'excuse" : 'Justifier une absence'}
                <Typography variant="body2" color="text.secondary">
                    {eleve ?? `Étudiant #${userId}`}
                    {enModification && ` · remplace la version ${formatPlageFR(justification!.starts_at, justification!.ends_at)}`}
                </Typography>
            </DialogTitle>

            <DialogContent dividers>
                {/* Aucun champ de motif : voir types.ts. */}
                <Box sx={{ display: 'flex', gap: 2, mb: 2.5, flexWrap: 'wrap' }}>
                    <TextField
                        label="Début" type="datetime-local" size="small" value={debut}
                        onChange={e => setDebut(e.target.value)}
                        InputLabelProps={{ shrink: true }}
                    />
                    <TextField
                        label="Fin" type="datetime-local" size="small" value={fin}
                        onChange={e => setFin(e.target.value)}
                        InputLabelProps={{ shrink: true }}
                    />
                    {isFetching && <CircularProgress size={20} sx={{ alignSelf: 'center' }} />}
                </Box>

                {!plageValide && (
                    <Alert severity="info">
                        Saisissez un début et une fin pour voir les séances concernées.
                    </Alert>
                )}

                {/* Chevauchement : rien ne l'interdit en base, mais deux excuses
                    concurrentes sur la même séance rendent une annulation sans
                    effet visible. Mieux vaut modifier l'existante. */}
                {(apercu?.chevauchements.length ?? 0) > 0 && (
                    <Alert severity="warning" sx={{ mb: 2 }}>
                        <AlertTitle>Une excuse couvre déjà cette période</AlertTitle>
                        {apercu!.chevauchements.map(c => (
                            <Box key={c.id}>
                                <Link
                                    component="button"
                                    type="button"
                                    onClick={() => onOuvrirAutre?.(c.id)}
                                    underline="hover"
                                >
                                    {formatPlageFR(c.starts_at, c.ends_at)}
                                </Link>
                            </Box>
                        ))}
                        Préférez modifier cette excuse plutôt qu'en créer une seconde : deux excuses
                        superposées se confondent sur la feuille de présence, et annuler l'une n'y
                        change alors rien.
                    </Alert>
                )}

                {plageValide && (
                    <>
                        <Typography variant="subtitle2" fontWeight={700} mb={0.5}>
                            Séances concernées
                        </Typography>
                        <Paper variant="outlined" sx={{ maxHeight: 320, overflow: 'auto' }}>
                            {seances.length === 0 ? (
                                <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                                    Aucune séance planifiée sur cette période. L'excuse est enregistrable :
                                    les séances ajoutées ensuite au planning y seront rattachées.
                                </Typography>
                            ) : (
                                <List dense disablePadding>
                                    {seances.map(s => {
                                        const retenue = s.dans_plage && !exclues.has(s.id);
                                        const entrante = enModification && s.dans_plage && !s.deja_couverte;
                                        const sortante = !s.dans_plage;
                                        const pointage = STATUT_POINTAGE[s.statut];
                                        return (
                                            <ListItemButton
                                                key={s.id}
                                                onClick={() => !sortante && toggle(s.id)}
                                                disabled={sortante}
                                                sx={sortante ? { opacity: 0.6 } : undefined}
                                            >
                                                <Checkbox
                                                    edge="start"
                                                    checked={retenue}
                                                    tabIndex={-1}
                                                    disableRipple
                                                    disabled={sortante}
                                                />
                                                <ListItemText
                                                    primary={s.matiere}
                                                    secondary={formatSeanceFR(s)}
                                                />
                                                <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', ml: 2 }}>
                                                    {entrante && <Chip size="small" color="success" variant="outlined" label="entre" />}
                                                    {sortante && <Chip size="small" variant="outlined" label="sort de la plage" />}
                                                    <Chip
                                                        size="small"
                                                        variant="outlined"
                                                        color={pointage?.color ?? 'default'}
                                                        label={pointage?.label ?? s.statut}
                                                    />
                                                </Box>
                                            </ListItemButton>
                                        );
                                    })}
                                </List>
                            )}
                        </Paper>
                    </>
                )}

                {confirmationRequise && (
                    <Alert severity="warning" sx={{ mt: 2 }}>
                        Cette excuse couvre {nbJours} jours et {retenues.length} séance{retenues.length > 1 ? 's' : ''}.
                        Confirmer ?
                    </Alert>
                )}

                {erreur && <Alert severity="error" sx={{ mt: 2 }}>{erreur}</Alert>}
            </DialogContent>

            <Divider />
            <DialogActions sx={{ px: 3, py: 1.5, justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">
                    {retenues.length} séance{retenues.length > 1 ? 's' : ''} · {dureeHeures(retenues)} h
                </Typography>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button onClick={onClose} disabled={enCours}>Annuler</Button>
                    <Button
                        variant="contained"
                        onClick={valider}
                        disabled={!plageValide || enCours}
                        sx={{ bgcolor: '#4f46e5', '&:hover': { bgcolor: '#4338ca' } }}
                    >
                        {enCours ? <CircularProgress size={16} sx={{ mr: 1, color: '#fff' }} /> : null}
                        {confirmationRequise ? 'Confirmer' : enModification ? 'Enregistrer la modification' : "Enregistrer l'excuse"}
                    </Button>
                </Box>
            </DialogActions>
        </Dialog>
    );
}
