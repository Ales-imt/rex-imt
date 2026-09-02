import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import dayjs from 'dayjs';
import {
    Alert, Box, Chip, IconButton, LinearProgress, MenuItem, Paper, Select,
    Skeleton, Stack, Table, TableBody, TableCell, TableHead, TableRow,
    TableSortLabel, TextField, Tooltip, Typography,
} from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import { apiInstance } from '../../services/api';
import { AMPLITUDES, labelSemaine, lundiDe } from '../../services/calendrier';
import { useSessionState } from '../../hooks/useSessionState';
import {
    ENDPOINT_SALLES, SALLE_SEMAINE_LUNDI_KEY, SALLE_SEMAINE_WORKFLOW,
    type OccupationSalle,
} from './def';

type CleTri = 'salle' | 'type' | 'capacite' | 'nb_seances' | 'heures';
type Ordre = 'asc' | 'desc';

export function Occupation() {
    const navigate = useNavigate();
    const [lundi, setLundi] = useSessionState('salle.occupation.lundi', lundiDe(dayjs()));
    const [amplitudeCle, setAmplitudeCle] = useSessionState<string>('salle.occupation.amplitude', AMPLITUDES[0].cle);
    const [recherche, setRecherche] = useSessionState<string>('salle.occupation.recherche', '');
    const [typeFiltre, setTypeFiltre] = useSessionState<string>('salle.occupation.type', '');

    const [triPar, setTriPar] = useState<CleTri>('heures');
    const [ordre, setOrdre] = useState<Ordre>('desc');

    const amplitude = (AMPLITUDES.find(a => a.cle === amplitudeCle) ?? AMPLITUDES[0]).heures;
    const fin = dayjs(lundi).add(7, 'day').format('YYYY-MM-DD');

    const occupationQ = useQuery<OccupationSalle[]>({
        queryKey: ['salle.occupation', lundi],
        queryFn: () => apiInstance
            .get(`${ENDPOINT_SALLES}/occupation?debut=${lundi}&fin=${fin}`)
            .then(r => r.data),
    });

    const types = useMemo(() => {
        const s = new Set<string>();
        for (const o of occupationQ.data ?? []) if (o.type) s.add(o.type);
        return [...s].sort();
    }, [occupationQ.data]);

    const trier = (cle: CleTri) => {
        if (triPar === cle) {
            setOrdre(ordre === 'asc' ? 'desc' : 'asc');
        } else {
            setTriPar(cle);
            setOrdre(cle === 'salle' || cle === 'type' ? 'asc' : 'desc');
        }
    };

    const rechercheMin = recherche.trim().toLowerCase();
    const lignes = useMemo(() => {
        const cmpNum = (a: number | null, b: number | null) =>
            (a ?? -1) - (b ?? -1);
        return (occupationQ.data ?? [])
            .filter(s => (!typeFiltre || s.type === typeFiltre)
                && (!rechercheMin || s.salle_name.toLowerCase().includes(rechercheMin)))
            .sort((a, b) => {
                let c: number;
                switch (triPar) {
                    case 'salle': c = a.salle_name.localeCompare(b.salle_name, 'fr'); break;
                    case 'type': c = (a.type ?? '').localeCompare(b.type ?? '', 'fr'); break;
                    case 'capacite': c = cmpNum(a.capacite, b.capacite); break;
                    case 'nb_seances': c = a.nb_seances - b.nb_seances; break;
                    default: c = a.heures - b.heures;
                }
                if (c === 0) c = a.salle_name.localeCompare(b.salle_name, 'fr');
                return ordre === 'asc' ? c : -c;
            });
    }, [occupationQ.data, typeFiltre, rechercheMin, triPar, ordre]);

    // Ouvre l'écran Semaine sur la même semaine que celle observée ici : sa
    // clé de session est écrite avant la navigation.
    const ouvrirSemaine = (salleId: number) => {
        sessionStorage.setItem(SALLE_SEMAINE_LUNDI_KEY, lundi);
        navigate(`/${SALLE_SEMAINE_WORKFLOW}/${salleId}`);
    };

    const totalHeures = lignes.reduce((s, l) => s + l.heures, 0);
    const tauxMoyen = lignes.length > 0 ? totalHeures / amplitude / lignes.length : 0;
    const inutilisees = lignes.filter(l => l.heures === 0).length;

    const colonnes: { cle: CleTri; label: string; align?: 'right' }[] = [
        { cle: 'salle', label: 'Salle' },
        { cle: 'type', label: 'Type' },
        { cle: 'capacite', label: 'Places', align: 'right' },
        { cle: 'nb_seances', label: 'Séances', align: 'right' },
        { cle: 'heures', label: 'Heures', align: 'right' },
    ];

    return (
        <Box sx={{ p: 2, maxWidth: 1100, mx: 'auto' }}>
            {/* Semaine et amplitude */}
            <Stack direction="row" alignItems="center" spacing={1} flexWrap="wrap" useFlexGap sx={{ mb: 2 }}>
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
                <Box sx={{ flex: 1 }} />
                <Select
                    size="small"
                    value={amplitudeCle}
                    onChange={e => setAmplitudeCle(e.target.value)}
                    sx={{ minWidth: 190 }}
                >
                    {AMPLITUDES.map(a => (
                        <MenuItem key={a.cle} value={a.cle}>{a.label} ({a.heures} h)</MenuItem>
                    ))}
                </Select>
            </Stack>

            {/* Filtres */}
            <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
                <TextField
                    size="small"
                    label="Salle"
                    value={recherche}
                    onChange={e => setRecherche(e.target.value)}
                    sx={{ minWidth: 220 }}
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
            </Stack>

            {occupationQ.isError && <Alert severity="error">Impossible de charger l'occupation des salles.</Alert>}
            {occupationQ.isLoading && <Skeleton variant="rounded" height={320} />}

            {occupationQ.isSuccess && occupationQ.data.length === 0 && (
                <Alert severity="info">
                    Aucune salle au référentiel : la synchronisation n'a pas encore importé les salles.
                </Alert>
            )}

            {occupationQ.isSuccess && occupationQ.data.length > 0 && (
                <>
                    {/* Synthèse */}
                    <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
                        {[
                            { titre: 'Taux moyen', valeur: `${(tauxMoyen * 100).toFixed(0)} %` },
                            { titre: 'Heures réservées', valeur: `${totalHeures.toFixed(1)} h` },
                            { titre: 'Salles inutilisées', valeur: `${inutilisees} / ${lignes.length}` },
                        ].map(carte => (
                            <Paper key={carte.titre} variant="outlined" sx={{ p: 2, flex: 1 }}>
                                <Typography variant="caption" color="text.secondary">{carte.titre}</Typography>
                                <Typography variant="h5">{carte.valeur}</Typography>
                            </Paper>
                        ))}
                    </Stack>

                    {occupationQ.data.every(s => s.heures === 0) && (
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Aucune réservation sur cette semaine.
                        </Alert>
                    )}

                    <Paper variant="outlined">
                        <Table size="small">
                            <TableHead>
                                <TableRow>
                                    {colonnes.map(col => (
                                        <TableCell key={col.cle} align={col.align} sortDirection={triPar === col.cle ? ordre : false}>
                                            <TableSortLabel
                                                active={triPar === col.cle}
                                                direction={triPar === col.cle ? ordre : 'asc'}
                                                onClick={() => trier(col.cle)}
                                            >
                                                {col.label}
                                            </TableSortLabel>
                                        </TableCell>
                                    ))}
                                    <TableCell sx={{ width: 220 }}>Occupation</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {lignes.map(salle => {
                                    const taux = salle.heures / amplitude;
                                    return (
                                        <TableRow
                                            key={salle.salle_id}
                                            hover
                                            sx={{ cursor: 'pointer' }}
                                            onClick={() => ouvrirSemaine(salle.salle_id)}
                                        >
                                            <TableCell>
                                                <Tooltip title="Voir la semaine de cette salle">
                                                    <Stack direction="row" alignItems="center" spacing={1}>
                                                        <span>{salle.salle_name}</span>
                                                        {salle.heures === 0 && (
                                                            <Chip size="small" variant="outlined" label="jamais réservée" />
                                                        )}
                                                    </Stack>
                                                </Tooltip>
                                            </TableCell>
                                            <TableCell>{salle.type ?? ''}</TableCell>
                                            <TableCell align="right">{salle.capacite ?? '—'}</TableCell>
                                            <TableCell align="right">{salle.nb_seances}</TableCell>
                                            <TableCell align="right">{salle.heures.toFixed(1)}</TableCell>
                                            <TableCell>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <LinearProgress
                                                        variant="determinate"
                                                        // Seule la LARGEUR est bornée : un taux > 100 %
                                                        // (séances qui se chevauchent) s'affiche tel quel.
                                                        value={Math.min(100, taux * 100)}
                                                        color={taux > 1 ? 'error' : 'primary'}
                                                        sx={{ flex: 1, height: 6, borderRadius: 3 }}
                                                    />
                                                    <Typography variant="caption" sx={{ width: 42, textAlign: 'right' }}>
                                                        {(taux * 100).toFixed(0)} %
                                                    </Typography>
                                                </Stack>
                                            </TableCell>
                                        </TableRow>
                                    );
                                })}
                                {lignes.length === 0 && (
                                    <TableRow>
                                        <TableCell colSpan={6}>
                                            <Typography variant="body2" color="text.secondary">
                                                Aucune salle ne correspond aux filtres.
                                            </Typography>
                                        </TableCell>
                                    </TableRow>
                                )}
                            </TableBody>
                        </Table>
                    </Paper>
                </>
            )}
        </Box>
    );
}
