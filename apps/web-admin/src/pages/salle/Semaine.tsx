import { useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';
import dayjs from 'dayjs';
import FullCalendar from '@fullcalendar/react';
import timeGridPlugin from '@fullcalendar/timegrid';
import dayGridPlugin from '@fullcalendar/daygrid';
import frLocale from '@fullcalendar/core/locales/fr';
import {
    Alert, Autocomplete, Box, MenuItem, Select, Skeleton, Stack, TextField,
    Typography,
} from '@mui/material';
import { apiInstance } from '../../services/api';
import { AMPLITUDES, lundiDe } from '../../services/calendrier';
import { useSessionState } from '../../hooks/useSessionState';
import {
    ENDPOINT_SALLES, SALLE_SEMAINE_LUNDI_KEY, SALLE_SEMAINE_SALLE_KEY,
    SALLE_SEMAINE_WORKFLOW, libelleOccupant,
    type CreneauSalle, type OccupationSalle,
} from './def';

// Une seule couleur : toutes les réservations concernent la même salle, une
// coloration n'apporterait aucune information que le libellé ne donne déjà.
const COULEUR_EVENEMENT = '#1976d2';

// L'écran d'invite et l'écran avec salle sont deux routes : en passer d'une à
// l'autre remonte le composant. Sans staleTime, React Query relancerait les
// requêtes à ce remontage et la première sélection de salle coûterait un
// appel réseau — le critère est zéro appel au changement de salle.
const STALE_MS = 5 * 60 * 1000;

function fmtHeures(h: number): string {
    return Number.isInteger(h) ? `${h} h` : `${h.toFixed(1)} h`;
}

export function Semaine() {
    const { salleId: salleIdParam } = useParams();
    const navigate = useNavigate();
    const salleId = salleIdParam ? Number(salleIdParam) : null;

    const [lundi, setLundi] = useSessionState(SALLE_SEMAINE_LUNDI_KEY, lundiDe(dayjs()));
    const [amplitudeCle, setAmplitudeCle] = useSessionState<string>('salle.semaine.amplitude', AMPLITUDES[0].cle);
    const amplitude = AMPLITUDES.find(a => a.cle === amplitudeCle) ?? AMPLITUDES[0];

    // Mémorise la salle ouverte ; l'entrée de menu (route sans salle) rouvre
    // la dernière consultée.
    useEffect(() => {
        if (salleIdParam) {
            sessionStorage.setItem(SALLE_SEMAINE_SALLE_KEY, salleIdParam);
        } else {
            const derniere = sessionStorage.getItem(SALLE_SEMAINE_SALLE_KEY);
            if (derniere) navigate(`/${SALLE_SEMAINE_WORKFLOW}/${derniere}`, { replace: true });
        }
    }, [salleIdParam, navigate]);

    const fin = dayjs(lundi).add(7, 'day').format('YYYY-MM-DD');

    // Toute la semaine, toutes salles confondues : le filtre par salle se fait
    // en mémoire, pour que comparer plusieurs salles ne coûte aucun réseau.
    const creneauxQ = useQuery<CreneauSalle[]>({
        queryKey: ['salle.semaine.creneaux', lundi],
        queryFn: () => apiInstance
            .get(`${ENDPOINT_SALLES}/creneaux?debut=${lundi}&fin=${fin}`)
            .then(r => r.data),
        staleTime: STALE_MS,
    });

    // Même clé que l'écran Occupation : même endpoint, mêmes paramètres — le
    // cache est partagé, arriver depuis Occupation ne recharge rien.
    const occupationQ = useQuery<OccupationSalle[]>({
        queryKey: ['salle.occupation', lundi],
        queryFn: () => apiInstance
            .get(`${ENDPOINT_SALLES}/occupation?debut=${lundi}&fin=${fin}`)
            .then(r => r.data),
        staleTime: STALE_MS,
    });

    // Triées par type puis nom : groupBy exige des options déjà groupées.
    const salles = useMemo(() =>
        [...(occupationQ.data ?? [])].sort((a, b) => {
            const t = (a.type ?? '').localeCompare(b.type ?? '', 'fr');
            return t !== 0 ? t : a.salle_name.localeCompare(b.salle_name, 'fr');
        }),
    [occupationQ.data]);

    const salleCourante = salles.find(s => s.salle_id === salleId) ?? null;

    // toEvent() de Planning.tsx n'est pas réutilisable ici : il travaille sur
    // ReservationDetail, dont aucun champ n'existe dans CreneauSalle.
    const evenements = useMemo(() =>
        (creneauxQ.data ?? [])
            .filter(c => c.salle_id === salleId)
            .map(c => ({
                id: String(c.seance_id),
                title: libelleOccupant(c),
                start: c.starts_at,
                end: c.ends_at,
            })),
    [creneauxQ.data, salleId]);

    const chargement = occupationQ.isLoading || creneauxQ.isLoading;
    const erreur = occupationQ.isError || creneauxQ.isError;
    const referentielVide = occupationQ.isSuccess && occupationQ.data.length === 0;

    return (
        <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', p: 2 }}>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ mb: 2 }}>
                <Autocomplete
                    size="small"
                    options={salles}
                    value={salleCourante}
                    onChange={(_, s) => navigate(s
                        ? `/${SALLE_SEMAINE_WORKFLOW}/${s.salle_id}`
                        : `/${SALLE_SEMAINE_WORKFLOW}`)}
                    groupBy={s => s.type ?? 'Sans type'}
                    getOptionLabel={s => s.salle_name}
                    isOptionEqualToValue={(a, b) => a.salle_id === b.salle_id}
                    renderOption={(props, s) => (
                        <li {...props} key={s.salle_id}>
                            <Box>
                                <Typography variant="body2">
                                    {[
                                        s.salle_name,
                                        s.type,
                                        s.capacite !== null ? `${s.capacite} pl.` : null,
                                        s.heures > 0 ? fmtHeures(s.heures) : null,
                                    ].filter(Boolean).join(' · ')}
                                </Typography>
                                {s.heures === 0 && (
                                    <Typography variant="caption" color="text.secondary">
                                        jamais réservée cette semaine
                                    </Typography>
                                )}
                            </Box>
                        </li>
                    )}
                    renderInput={p => <TextField {...p} label="Salle" />}
                    sx={{ minWidth: 320 }}
                />
                <Box sx={{ flex: 1 }} />
                <Select
                    size="small"
                    value={amplitudeCle}
                    onChange={e => setAmplitudeCle(e.target.value)}
                    sx={{ minWidth: 190 }}
                >
                    {AMPLITUDES.map(a => (
                        <MenuItem key={a.cle} value={a.cle}>{a.label}</MenuItem>
                    ))}
                </Select>
            </Stack>

            {erreur && <Alert severity="error">Impossible de charger les réservations des salles.</Alert>}
            {chargement && <Skeleton variant="rounded" height={420} />}

            {referentielVide && (
                <Alert severity="info">
                    Aucune salle au référentiel : la synchronisation n'a pas encore importé les salles.
                </Alert>
            )}

            {!chargement && !erreur && !referentielVide && salleId === null && (
                <Alert severity="info">
                    Choisissez une salle pour afficher ses réservations de la semaine.
                </Alert>
            )}

            {!chargement && !erreur && !referentielVide && salleId !== null && (
                <>
                    {occupationQ.isSuccess && !salleCourante && (
                        <Alert severity="warning" sx={{ mb: 1 }}>
                            Salle inconnue du référentiel.
                        </Alert>
                    )}
                    {creneauxQ.isSuccess && salleCourante && evenements.length === 0 && (
                        // La grille reste affichée : bornée par l'amplitude,
                        // une semaine vide EST une réponse — la salle est libre.
                        <Alert severity="info" sx={{ mb: 1 }}>
                            Aucune réservation cette semaine.
                        </Alert>
                    )}
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                        <FullCalendar
                            plugins={[timeGridPlugin, dayGridPlugin]}
                            initialView="timeGridWeek"
                            initialDate={lundi}
                            datesSet={info => setLundi(lundiDe(dayjs(info.start)))}
                            locale={frLocale}
                            headerToolbar={{ left: 'prev,next today', center: 'title', right: '' }}
                            events={evenements}
                            eventColor={COULEUR_EVENEMENT}
                            allDaySlot={false}
                            // Bornes par l'amplitude choisie : l'utilisateur
                            // cherche les trous UTILISABLES ; un trou hors
                            // amplitude n'est pas un créneau, l'afficher
                            // diluerait l'information.
                            slotMinTime={amplitude.debut}
                            slotMaxTime={amplitude.fin}
                            hiddenDays={amplitude.samedi ? [0] : [0, 6]}
                            height="100%"
                            selectable={false}
                            editable={false}
                            eventContent={info => (
                                <Box sx={{ fontSize: '0.75rem', overflow: 'hidden', px: 0.5, lineHeight: 1.3 }}>
                                    <div>{info.timeText}</div>
                                    <strong>{info.event.title}</strong>
                                </Box>
                            )}
                        />
                    </Box>
                </>
            )}
        </Box>
    );
}
