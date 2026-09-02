import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import {
    Alert, Box, Chip, IconButton, MenuItem, Paper, Select, Skeleton, Stack,
    Slider, TextField, ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import MyLocationIcon from '@mui/icons-material/MyLocation';
import { apiInstance } from '../../services/api';
import { useSessionState } from '../../hooks/useSessionState';
import {
    ENDPOINT_SALLES, fmtHeure, libelleOccupant, statutSalle,
    type CreneauSalle, type OccupationSalle,
} from './def';

const H_DEBUT = 8 * 60;
const H_FIN = 20 * 60;

type FiltreStatut = 'toutes' | 'libres' | 'occupees';

function labelJour(jour: string): string {
    const d = new Date(jour + 'T12:00:00');
    return d.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' });
}

function fmtMinutes(m: number): string {
    return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;
}

export function Disponibilite() {
    const aujourdhui = dayjs().format('YYYY-MM-DD');
    const [jour, setJour] = useSessionState('salle.dispo.jour', aujourdhui);
    const [minutesStr, setMinutesStr] = useSessionState('salle.dispo.minutes', String(H_DEBUT));
    // « Maintenant » : l'utilisateur n'a pas touché au curseur, l'écran suit
    // l'heure courante et se rafraîchit. Dès qu'il le déplace, il regarde une
    // projection — plus de rafraîchissement automatique.
    const [maintenant, setMaintenant] = useSessionState<string>('salle.dispo.maintenant', '1');

    const [recherche, setRecherche] = useSessionState<string>('salle.dispo.recherche', '');
    const [typeFiltre, setTypeFiltre] = useSessionState<string>('salle.dispo.type', '');
    const [capMinStr, setCapMinStr] = useSessionState<string>('salle.dispo.capmin', '');
    const [filtreStatut, setFiltreStatut] = useSessionState<FiltreStatut>('salle.dispo.statut', 'toutes');

    const enModeMaintenant = maintenant === '1';
    const finJour = dayjs(jour).add(1, 'day').format('YYYY-MM-DD');

    // La JOURNÉE entière est chargée, jamais la plage depuis l'instant
    // observé : /creneaux filtre sur starts_at >= debut, demander depuis 14h
    // perdrait toutes les séances déjà commencées — exactement celles qui
    // occupent les salles. Le curseur ne recalcule qu'en mémoire.
    const occupationQ = useQuery<OccupationSalle[]>({
        queryKey: ['salle.dispo.occupation', jour],
        queryFn: () => apiInstance
            .get(`${ENDPOINT_SALLES}/occupation?debut=${jour}&fin=${finJour}`)
            .then(r => r.data),
        refetchInterval: enModeMaintenant ? 5 * 60 * 1000 : false,
    });
    const creneauxQ = useQuery<CreneauSalle[]>({
        queryKey: ['salle.dispo.creneaux', jour],
        queryFn: () => apiInstance
            .get(`${ENDPOINT_SALLES}/creneaux?debut=${jour}&fin=${finJour}`)
            .then(r => r.data),
        refetchInterval: enModeMaintenant ? 5 * 60 * 1000 : false,
    });

    const nowMinutes = dayjs().hour() * 60 + dayjs().minute();
    const minutes = enModeMaintenant
        ? Math.min(H_FIN, Math.max(H_DEBUT, nowMinutes - (nowMinutes % 15)))
        : Number(minutesStr);
    const t = enModeMaintenant ? dayjs() : dayjs(jour).startOf('day').add(minutes, 'minute');

    const bougerCurseur = (m: number) => {
        setMaintenant('0');
        setMinutesStr(String(m));
    };
    const changerJour = (delta: number) => {
        setMaintenant('0');
        setJour(dayjs(jour).add(delta, 'day').format('YYYY-MM-DD'));
    };
    const revenirMaintenant = () => {
        setJour(dayjs().format('YYYY-MM-DD'));
        setMaintenant('1');
    };

    const creneauxParSalle = useMemo(() => {
        const m = new Map<number, CreneauSalle[]>();
        for (const c of creneauxQ.data ?? []) {
            const l = m.get(c.salle_id);
            if (l) l.push(c); else m.set(c.salle_id, [c]);
        }
        return m;
    }, [creneauxQ.data]);

    const types = useMemo(() => {
        const s = new Set<string>();
        for (const o of occupationQ.data ?? []) if (o.type) s.add(o.type);
        return [...s].sort();
    }, [occupationQ.data]);

    const capMin = Number(capMinStr) || 0;
    const rechercheMin = recherche.trim().toLowerCase();

    const lignes = useMemo(() => {
        let masqueesCapaciteInconnue = 0;
        const resultat = (occupationQ.data ?? [])
            .filter(salle => {
                if (typeFiltre && salle.type !== typeFiltre) return false;
                if (capMin > 0) {
                    // capacite null = « on ne sait pas », pas « 0 place » : un
                    // filtre de capacité doit exclure la salle, mais l'exclusion
                    // est comptée et annoncée sous la liste.
                    if (salle.capacite === null) {
                        masqueesCapaciteInconnue++;
                        return false;
                    }
                    if (salle.capacite < capMin) return false;
                }
                if (rechercheMin) {
                    const creneaux = creneauxParSalle.get(salle.salle_id) ?? [];
                    const textes = [
                        salle.salle_name,
                        ...creneaux.flatMap(c => [c.matiere_name, c.prof ?? '', c.groupe_name ?? '', c.promotion_name ?? '']),
                    ];
                    if (!textes.some(x => x.toLowerCase().includes(rechercheMin))) return false;
                }
                return true;
            })
            .map(salle => ({
                salle,
                statut: statutSalle(creneauxParSalle.get(salle.salle_id) ?? [], t),
            }))
            .sort((a, b) => {
                const occA = a.statut.occupants.length > 0 ? 1 : 0;
                const occB = b.statut.occupants.length > 0 ? 1 : 0;
                if (occA !== occB) return occA - occB;
                return a.salle.salle_name.localeCompare(b.salle.salle_name, 'fr');
            });
        return { resultat, masqueesCapaciteInconnue };
    }, [occupationQ.data, creneauxParSalle, typeFiltre, capMin, rechercheMin, t]);

    const nbLibres = lignes.resultat.filter(l => l.statut.occupants.length === 0).length;
    const nbOccupees = lignes.resultat.length - nbLibres;
    const visibles = lignes.resultat.filter(l =>
        filtreStatut === 'toutes'
        || (filtreStatut === 'libres' && l.statut.occupants.length === 0)
        || (filtreStatut === 'occupees' && l.statut.occupants.length > 0));

    const chargement = occupationQ.isLoading || creneauxQ.isLoading;
    const erreur = occupationQ.isError || creneauxQ.isError;

    return (
        <Box sx={{ p: 2, maxWidth: 1000, mx: 'auto' }}>
            {/* Jour et instant observés */}
            <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 1 }}>
                    <IconButton size="small" onClick={() => changerJour(-1)} aria-label="jour précédent">
                        <ChevronLeftIcon />
                    </IconButton>
                    <Typography variant="subtitle1" sx={{ minWidth: 220, textAlign: 'center', textTransform: 'capitalize' }}>
                        {labelJour(jour)}
                    </Typography>
                    <IconButton size="small" onClick={() => changerJour(1)} aria-label="jour suivant">
                        <ChevronRightIcon />
                    </IconButton>
                    <Box sx={{ flex: 1 }} />
                    <Chip
                        icon={<MyLocationIcon />}
                        label="Maintenant"
                        color={enModeMaintenant ? 'primary' : 'default'}
                        variant={enModeMaintenant ? 'filled' : 'outlined'}
                        onClick={revenirMaintenant}
                    />
                </Stack>
                <Stack direction="row" alignItems="center" spacing={2}>
                    <Typography variant="body2" color="text.secondary" sx={{ minWidth: 44 }}>
                        {enModeMaintenant ? t.format('HH:mm') : fmtMinutes(minutes)}
                    </Typography>
                    <Slider
                        value={minutes}
                        min={H_DEBUT}
                        max={H_FIN}
                        step={15}
                        onChange={(_, v) => bougerCurseur(v as number)}
                        valueLabelDisplay="auto"
                        valueLabelFormat={fmtMinutes}
                        marks={[8, 10, 12, 14, 16, 18, 20].map(h => ({ value: h * 60, label: `${h}h` }))}
                    />
                </Stack>
            </Paper>

            {/* Filtres */}
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ mb: 2 }}>
                <TextField
                    size="small"
                    label="Recherche"
                    placeholder="Salle, prof, promo, groupe, matière…"
                    value={recherche}
                    onChange={e => setRecherche(e.target.value)}
                    sx={{ minWidth: 260 }}
                />
                <Select
                    size="small"
                    value={typeFiltre}
                    onChange={e => setTypeFiltre(e.target.value)}
                    displayEmpty
                    sx={{ minWidth: 160 }}
                >
                    <MenuItem value="">Tous les types</MenuItem>
                    {types.map(ty => <MenuItem key={ty} value={ty}>{ty}</MenuItem>)}
                </Select>
                <TextField
                    size="small"
                    label="Places min."
                    type="number"
                    value={capMinStr}
                    onChange={e => setCapMinStr(e.target.value)}
                    sx={{ width: 110 }}
                    slotProps={{ htmlInput: { min: 0 } }}
                />
                <ToggleButtonGroup
                    size="small"
                    exclusive
                    value={filtreStatut}
                    onChange={(_, v) => { if (v) setFiltreStatut(v); }}
                >
                    <ToggleButton value="toutes" sx={{ textTransform: 'none' }}>Toutes</ToggleButton>
                    <ToggleButton value="libres" sx={{ textTransform: 'none' }}>Libres</ToggleButton>
                    <ToggleButton value="occupees" sx={{ textTransform: 'none' }}>Occupées</ToggleButton>
                </ToggleButtonGroup>
            </Stack>

            {erreur && <Alert severity="error">Impossible de charger les salles.</Alert>}
            {chargement && (
                <Stack spacing={1}>
                    {[...Array(8)].map((_, i) => <Skeleton key={i} variant="rounded" height={52} />)}
                </Stack>
            )}

            {!chargement && !erreur && (occupationQ.data?.length ?? 0) === 0 && (
                <Alert severity="info">
                    Aucune salle au référentiel : la synchronisation n'a pas encore importé les salles.
                </Alert>
            )}

            {!chargement && !erreur && (occupationQ.data?.length ?? 0) > 0 && (
                <Paper variant="outlined">
                    <Box sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider' }}>
                        <Typography variant="subtitle2">
                            {nbLibres} libre{nbLibres > 1 ? 's' : ''} · {nbOccupees} occupée{nbOccupees > 1 ? 's' : ''}
                        </Typography>
                    </Box>

                    {visibles.map(({ salle, statut }) => {
                        const occupee = statut.occupants.length > 0;
                        const finOccupation = occupee
                            ? statut.occupants.reduce((max, c) => dayjs(c.ends_at).isAfter(max) ? dayjs(c.ends_at) : max, dayjs(statut.occupants[0].ends_at))
                            : null;
                        return (
                            <Stack
                                key={salle.salle_id}
                                direction="row"
                                alignItems="center"
                                spacing={1.5}
                                sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}
                            >
                                <Chip
                                    size="small"
                                    label={occupee ? 'Occupée' : 'Libre'}
                                    color={occupee ? 'warning' : 'success'}
                                    sx={{ minWidth: 76 }}
                                />
                                <Box sx={{ flex: 1, minWidth: 0 }}>
                                    <Stack direction="row" alignItems="baseline" spacing={1}>
                                        <Typography variant="body2" fontWeight={600} noWrap>
                                            {salle.salle_name}
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary" noWrap>
                                            {[salle.type, salle.capacite !== null ? `${salle.capacite} pl.` : null]
                                                .filter(Boolean).join(' · ')}
                                        </Typography>
                                        {statut.occupants.length > 1 && (
                                            <Tooltip title="Plusieurs séances se chevauchent dans cette salle : double réservation en amont.">
                                                <Chip size="small" variant="outlined" color="warning" label="chevauchement" />
                                            </Tooltip>
                                        )}
                                    </Stack>
                                    <Typography variant="caption" color="text.secondary" noWrap component="div">
                                        {occupee
                                            ? statut.occupants.map(libelleOccupant).join(' — ')
                                            : statut.prochain
                                                ? `Libre jusqu'à ${fmtHeure(statut.prochain.starts_at)}, puis ${libelleOccupant(statut.prochain)}`
                                                : 'Libre le reste de la journée'}
                                    </Typography>
                                </Box>
                                {occupee && finOccupation && (
                                    <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'nowrap' }}>
                                        jusqu'à {finOccupation.format('HH:mm')}
                                    </Typography>
                                )}
                            </Stack>
                        );
                    })}

                    {visibles.length === 0 && (
                        <Typography variant="body2" color="text.secondary" sx={{ px: 2, py: 2 }}>
                            Aucune salle ne correspond aux filtres.
                        </Typography>
                    )}

                    {lignes.masqueesCapaciteInconnue > 0 && (
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', px: 2, py: 1 }}>
                            {lignes.masqueesCapaciteInconnue} salle{lignes.masqueesCapaciteInconnue > 1 ? 's' : ''} à
                            capacité inconnue masquée{lignes.masqueesCapaciteInconnue > 1 ? 's' : ''} par le filtre.
                        </Typography>
                    )}
                </Paper>
            )}
        </Box>
    );
}
