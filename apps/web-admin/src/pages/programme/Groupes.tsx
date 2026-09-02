import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';
import dayjs from 'dayjs';
import {
    Alert, Box, Chip, IconButton, MenuItem, Paper, Select, Skeleton, Stack,
    ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import { apiInstance } from '../../services/api';
import { AMPLITUDES, labelSemaine, lundiDe } from '../../services/calendrier';
import { useSessionState } from '../../hooks/useSessionState';
import { PROGRAMME_WORKFLOW, comparerGroupes } from './def';
import type { GroupeDetail, ReservationDetail } from './def';

// ─── Modèle de la grille ──────────────────────────────────────────────────────

type Mode = 'semaine' | 'jour';

const COULEUR_GROUPE = '#1976d2';
const HAUTEUR_LIGNE = 30;
const LARGEUR_LIBELLE = 200;

interface Jour {
    date: string;  // YYYY-MM-DD
    label: string; // « lun. 7 sept. »
}

// Un bloc positionné sur une ligne : la réservation, sa place en pourcentage
// de l'amplitude, et ce qu'il représente.
interface Bloc {
    r: ReservationDetail;
    jour: string;
    left: number;
    width: number;
    debutMin: number;
    finMin: number;
    promo: boolean;   // bande de promotion (CM en amphi), peinte sur chaque ligne
    conflit: boolean; // chevauche un autre bloc de la même ligne
}

interface Ligne {
    groupe: GroupeDetail;
    blocs: Bloc[];
    heures: number;   // heures PROPRES au groupe, hors bandes de promotion
    conflits: number; // paires de blocs qui se chevauchent sur cette ligne
}

function minutesDe(hms: string): number {
    const [h, m] = hms.split(':').map(Number);
    return h * 60 + m;
}

function fmtHeures(h: number): string {
    return Number.isInteger(h) ? `${h} h` : `${h.toFixed(1)} h`;
}

// Place une réservation dans l'amplitude d'un jour ; null si elle tombe
// entièrement hors amplitude — un créneau hors amplitude n'est pas un créneau.
function placer(r: ReservationDetail, ampDebut: number, ampFin: number, promo: boolean): Omit<Bloc, 'conflit'> | null {
    const debut = dayjs(r.horaire.Lower);
    const fin = dayjs(r.horaire.Upper);
    const jour = debut.format('YYYY-MM-DD');
    const debutMin = debut.hour() * 60 + debut.minute();
    // Une séance qui déborde sur le lendemain est bornée à la fin du jour.
    const finMin = fin.format('YYYY-MM-DD') === jour ? fin.hour() * 60 + fin.minute() : 24 * 60;
    const d = Math.max(debutMin, ampDebut);
    const f = Math.min(finMin, ampFin);
    if (f <= d) return null;
    const span = ampFin - ampDebut;
    return {
        r, jour, promo,
        debutMin: d, finMin: f,
        left: ((d - ampDebut) / span) * 100,
        width: ((f - d) / span) * 100,
    };
}

function libelleBloc(r: ReservationDetail): string {
    return [r.type_cours, r.matiere_name, r.salles.map(s => s.name).join(', ')].filter(Boolean).join(' · ');
}

function detailBloc(b: Bloc): string {
    const r = b.r;
    const profs = r.intervenants.map(i => [i.firstName, i.lastName].filter(Boolean).join(' ')).join(', ');
    return [
        `${dayjs(r.horaire.Lower).format('HH:mm')}–${dayjs(r.horaire.Upper).format('HH:mm')}`,
        b.promo ? `Toute la promotion${r.promotion_name ? ` ${r.promotion_name}` : ''}` : null,
        r.type_cours,
        r.matiere_name,
        profs || null,
        r.salles.map(s => s.name).join(', ') || null,
        r.remarque,
    ].filter(Boolean).join(' · ');
}

// Marque les blocs d'une ligne qui se chevauchent — un groupe convoqué à deux
// endroits en même temps — et compte les paires. Les bandes de promotion
// participent : un TD pendant l'amphi est un vrai conflit. Deux bandes entre
// elles ne comptent pas ici, la même paire se répéterait sur chaque ligne ;
// elles sont comptées une seule fois par l'appelant.
function marquerConflits(blocs: Bloc[], compterPromoEntreElles: boolean): number {
    let n = 0;
    for (let i = 0; i < blocs.length; i++) {
        for (let j = i + 1; j < blocs.length; j++) {
            const a = blocs[i], b = blocs[j];
            if (a.jour !== b.jour) continue;
            if (a.debutMin >= b.finMin || b.debutMin >= a.finMin) continue;
            a.conflit = b.conflit = true;
            if (compterPromoEntreElles || !(a.promo && b.promo)) n++;
        }
    }
    return n;
}

// ─── Composant ────────────────────────────────────────────────────────────────

export function Groupes() {
    const { periodeId } = useParams();
    const navigate = useNavigate();

    const [lundi, setLundi] = useSessionState('programme.groupes.lundi', lundiDe(dayjs()));
    const [amplitudeCle, setAmplitudeCle] = useSessionState<string>('programme.groupes.amplitude', AMPLITUDES[0].cle);
    const [mode, setMode] = useSessionState<Mode>('programme.groupes.mode', 'semaine');
    const [jourStr, setJourStr] = useSessionState<string>('programme.groupes.jour', '0');
    const amplitude = AMPLITUDES.find(a => a.cle === amplitudeCle) ?? AMPLITUDES[0];
    const ampDebut = minutesDe(amplitude.debut);
    const ampFin = minutesDe(amplitude.fin);

    // Même clé que Planning.tsx : le cache est partagé entre les deux vues.
    const reservationsQ = useQuery<ReservationDetail[]>({
        queryKey: ['programme.reservations', periodeId],
        queryFn: () => apiInstance.get(`/api/v2/planning/reservation?periode_id=${periodeId}`).then(r => r.data),
        enabled: !!periodeId,
    });
    const groupesQ = useQuery<GroupeDetail[]>({
        queryKey: ['programme.groupes', periodeId],
        queryFn: () => apiInstance.get(`/api/v2/planning/groupes?periode_id=${periodeId}`).then(r => r.data),
        enabled: !!periodeId,
    });

    const jours: Jour[] = useMemo(() => {
        const n = amplitude.samedi ? 6 : 5;
        return Array.from({ length: n }, (_, i) => {
            const d = dayjs(lundi).add(i, 'day');
            return { date: d.format('YYYY-MM-DD'), label: d.toDate().toLocaleDateString('fr-FR', { weekday: 'short', day: 'numeric', month: 'short' }) };
        });
    }, [lundi, amplitude.samedi]);

    // /reservation rend toute la période : la semaine se filtre en mémoire,
    // changer de semaine ne coûte aucun appel réseau.
    const semaine = useMemo(() => {
        const fin = dayjs(lundi).add(7, 'day').format('YYYY-MM-DD');
        return (reservationsQ.data ?? []).filter(r => {
            const j = dayjs(r.horaire.Lower).format('YYYY-MM-DD');
            return j >= lundi && j < fin;
        });
    }, [reservationsQ.data, lundi]);

    const grille = useMemo(() => {
        const groupes = [...(groupesQ.data ?? [])].sort((a, b) => comparerGroupes(a.groupe_name, b.groupe_name));
        const ids = new Set(groupes.map(g => g.groupe_id));
        const bandes: Bloc[] = [];
        // Séances qui ne tomberont sur aucune ligne : sans groupe ni promotion,
        // ou visant un groupe d'une autre promotion (période importée avec
        // des rattachements approximatifs). Les compter évite de croire la
        // grille exhaustive quand elle ne l'est pas.
        let horsGrille = 0;
        for (const r of semaine) {
            if (r.groupes.length > 0) {
                if (!r.groupes.some(g => ids.has(g.id))) horsGrille++;
                continue;
            }
            if (r.promotion_id == null) { horsGrille++; continue; }
            const p = placer(r, ampDebut, ampFin, true);
            if (p) bandes.push({ ...p, conflit: false });
        }
        // Deux amphis en même temps : compté une fois, pas une fois par ligne.
        const conflitsPromo = marquerConflits(bandes, true);

        const lignes: Ligne[] = groupes.map(groupe => {
            const propres: Bloc[] = [];
            let heures = 0;
            for (const r of semaine) {
                if (!r.groupes.some(g => g.id === groupe.groupe_id)) continue;
                heures += dayjs(r.horaire.Upper).diff(dayjs(r.horaire.Lower), 'minute') / 60;
                const p = placer(r, ampDebut, ampFin, false);
                if (p) propres.push({ ...p, conflit: false });
            }
            const blocs = [...bandes.map(b => ({ ...b, conflit: false })), ...propres];
            const conflits = marquerConflits(blocs, false);
            return { groupe, blocs, heures, conflits };
        });
        const conflits = conflitsPromo + lignes.reduce((n, l) => n + l.conflits, 0);
        return { lignes, horsGrille, conflits };
    }, [groupesQ.data, semaine, ampDebut, ampFin]);

    const jourIdx = Math.min(Number(jourStr) || 0, jours.length - 1);
    const joursAffiches = mode === 'jour' ? [jours[jourIdx]] : jours;
    const nbHeures = (ampFin - ampDebut) / 60;
    const pasTicks = mode === 'jour' ? 1 : 2;
    const ticks = Array.from({ length: Math.floor(nbHeures / pasTicks) + 1 }, (_, i) => ampDebut + i * pasTicks * 60)
        .filter(m => m < ampFin);

    const chargement = reservationsQ.isLoading || groupesQ.isLoading;
    const erreur = reservationsQ.isError || groupesQ.isError;
    const sansGroupe = groupesQ.isSuccess && groupesQ.data.length === 0;

    const ouvrirJour = (i: number) => { setJourStr(String(i)); setMode('jour'); };

    return (
        <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', p: 2, overflow: 'hidden' }}>
            <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap sx={{ mb: 2 }}>
                <IconButton size="small" onClick={() => setLundi(dayjs(lundi).subtract(7, 'day').format('YYYY-MM-DD'))} aria-label="semaine précédente">
                    <ChevronLeftIcon />
                </IconButton>
                <Typography variant="subtitle1" sx={{ minWidth: 230, textAlign: 'center' }}>
                    {labelSemaine(lundi)}
                </Typography>
                <IconButton size="small" onClick={() => setLundi(dayjs(lundi).add(7, 'day').format('YYYY-MM-DD'))} aria-label="semaine suivante">
                    <ChevronRightIcon />
                </IconButton>
                <Chip label="Aujourd'hui" variant="outlined" onClick={() => setLundi(lundiDe(dayjs()))} />

                <ToggleButtonGroup size="small" exclusive value={mode} onChange={(_, m: Mode | null) => { if (m) setMode(m); }}>
                    <ToggleButton value="semaine">Semaine</ToggleButton>
                    <ToggleButton value="jour">Jour</ToggleButton>
                </ToggleButtonGroup>
                {mode === 'jour' && (
                    <ToggleButtonGroup size="small" exclusive value={jourIdx} onChange={(_, i: number | null) => { if (i !== null) setJourStr(String(i)); }}>
                        {jours.map((j, i) => <ToggleButton key={j.date} value={i}>{j.label.split(' ')[0]}</ToggleButton>)}
                    </ToggleButtonGroup>
                )}

                {grille.conflits > 0 && (
                    <Tooltip title="Un groupe convoqué à deux endroits en même temps">
                        <Chip color="error" size="small" label={`${grille.conflits} chevauchement${grille.conflits > 1 ? 's' : ''}`} />
                    </Tooltip>
                )}
                {grille.horsGrille > 0 && (
                    <Tooltip title="Séances sans groupe ni promotion, ou visant un groupe d'une autre promotion : elles n'apparaissent sur aucune ligne">
                        <Chip color="warning" size="small" variant="outlined" label={`${grille.horsGrille} hors grille`} />
                    </Tooltip>
                )}

                <Box sx={{ flex: 1 }} />
                <Select size="small" value={amplitudeCle} onChange={e => setAmplitudeCle(e.target.value)} sx={{ minWidth: 190 }}>
                    {AMPLITUDES.map(a => <MenuItem key={a.cle} value={a.cle}>{a.label}</MenuItem>)}
                </Select>
                <Tooltip title="Vue calendrier">
                    <IconButton size="small" onClick={() => navigate(`/${PROGRAMME_WORKFLOW}/${periodeId}`)}>
                        <CalendarMonthIcon />
                    </IconButton>
                </Tooltip>
            </Stack>

            {erreur && <Alert severity="error">Impossible de charger le planning de la période.</Alert>}
            {chargement && <Skeleton variant="rounded" height={420} />}

            {sansGroupe && (
                <Alert severity="warning">
                    La promotion de cette période n'a aucun groupe au référentiel : rien à planifier par groupe.
                </Alert>
            )}

            {!chargement && !erreur && !sansGroupe && (
                <>
                    {reservationsQ.isSuccess && semaine.length === 0 && (
                        // La grille reste affichée : ici, une semaine vide EST
                        // une réponse — aucun groupe n'est planifié.
                        <Alert severity="info" sx={{ mb: 1 }}>Aucune séance cette semaine.</Alert>
                    )}
                    <Paper variant="outlined" sx={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
                        <Box sx={{
                            display: 'grid',
                            gridTemplateColumns: `${LARGEUR_LIBELLE}px repeat(${joursAffiches.length}, minmax(0, 1fr))`,
                            minWidth: mode === 'jour' ? 0 : 900,
                        }}>
                            {/* En-tête : jours, puis graduation horaire */}
                            <Box sx={{ position: 'sticky', top: 0, zIndex: 2, bgcolor: 'background.paper', borderBottom: 1, borderColor: 'divider' }} />
                            {joursAffiches.map((j, i) => (
                                <Box
                                    key={j.date}
                                    onClick={mode === 'semaine' ? () => ouvrirJour(i) : undefined}
                                    sx={{
                                        position: 'sticky', top: 0, zIndex: 2, bgcolor: 'background.paper',
                                        borderBottom: 1, borderLeft: 1, borderColor: 'divider',
                                        px: 1, pt: 0.5, cursor: mode === 'semaine' ? 'pointer' : 'default',
                                        '&:hover': mode === 'semaine' ? { bgcolor: 'action.hover' } : undefined,
                                    }}
                                >
                                    <Typography variant="subtitle2" noWrap>{j.label}</Typography>
                                    <Box sx={{ position: 'relative', height: 16 }}>
                                        {ticks.map(m => (
                                            <Typography
                                                key={m}
                                                variant="caption"
                                                color="text.secondary"
                                                sx={{ position: 'absolute', left: `${((m - ampDebut) / (ampFin - ampDebut)) * 100}%`, lineHeight: 1, fontSize: '0.65rem' }}
                                            >
                                                {m / 60}h
                                            </Typography>
                                        ))}
                                    </Box>
                                </Box>
                            ))}

                            {/* Une ligne par groupe du référentiel, planifié ou non */}
                            {grille.lignes.map(ligne => {
                                const vide = ligne.heures === 0;
                                return [
                                    <Box
                                        key={`l-${ligne.groupe.groupe_id}`}
                                        sx={{
                                            px: 1, height: HAUTEUR_LIGNE, display: 'flex', alignItems: 'center', gap: 1,
                                            borderBottom: 1, borderColor: 'divider', overflow: 'hidden',
                                            color: vide ? 'warning.main' : 'text.primary',
                                        }}
                                    >
                                        <Typography variant="body2" noWrap sx={{ fontWeight: vide ? 700 : 500, flex: 1 }}>
                                            {ligne.groupe.groupe_name}
                                        </Typography>
                                        <Typography variant="caption" noWrap color={vide ? 'warning.main' : 'text.secondary'}>
                                            {[
                                                ligne.groupe.taille ? `${ligne.groupe.taille} él.` : null,
                                                fmtHeures(ligne.heures),
                                            ].filter(Boolean).join(' · ')}
                                        </Typography>
                                    </Box>,
                                    ...joursAffiches.map(j => (
                                        <Box
                                            key={`${ligne.groupe.groupe_id}-${j.date}`}
                                            sx={{
                                                position: 'relative', height: HAUTEUR_LIGNE,
                                                borderBottom: 1, borderLeft: 1, borderColor: 'divider',
                                                bgcolor: vide ? 'rgba(237, 108, 2, 0.06)' : undefined,
                                                backgroundImage: `repeating-linear-gradient(to right, rgba(0,0,0,0.07) 0 1px, transparent 1px ${100 / nbHeures}%)`,
                                            }}
                                        >
                                            {ligne.blocs.filter(b => b.jour === j.date).map((b, k) => (
                                                <Tooltip key={`${b.r.id}-${k}`} title={detailBloc(b)} arrow>
                                                    <Box sx={{
                                                        position: 'absolute', top: 3, bottom: 3,
                                                        left: `${b.left}%`, width: `${b.width}%`,
                                                        boxSizing: 'border-box', borderRadius: 0.5,
                                                        zIndex: b.promo ? 0 : 1,
                                                        px: 0.5, overflow: 'hidden', whiteSpace: 'nowrap',
                                                        fontSize: '0.7rem', lineHeight: `${HAUTEUR_LIGNE - 6}px`,
                                                        ...(b.promo
                                                            ? { bgcolor: 'action.selected', color: 'text.secondary', border: '1px solid', borderColor: 'divider' }
                                                            : { bgcolor: COULEUR_GROUPE, color: '#fff' }),
                                                        ...(b.conflit ? { border: '2px solid', borderColor: 'error.main' } : {}),
                                                    }}>
                                                        {mode === 'jour' && libelleBloc(b.r)}
                                                    </Box>
                                                </Tooltip>
                                            ))}
                                        </Box>
                                    )),
                                ];
                            })}
                        </Box>
                    </Paper>
                </>
            )}
        </Box>
    );
}
