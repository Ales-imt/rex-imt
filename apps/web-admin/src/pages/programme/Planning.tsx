import FullCalendar from '@fullcalendar/react';
import timeGridPlugin from '@fullcalendar/timegrid';
import dayGridPlugin from '@fullcalendar/daygrid';
import frLocale from '@fullcalendar/core/locales/fr';
import type { EventClickArg, EventContentArg } from '@fullcalendar/core';
import { useQuery } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router';
import {
    Box, Divider, FormControl, IconButton, InputLabel, MenuItem, Popover, Select, Stack, Tooltip, Typography,
} from '@mui/material';
import { useState, useEffect, useCallback, useMemo } from 'react';
import dayjs from 'dayjs';
import MenuOpenIcon from '@mui/icons-material/MenuOpen';
import PaletteIcon from '@mui/icons-material/Palette';
import TuneIcon from '@mui/icons-material/Tune';
import ViewTimelineIcon from '@mui/icons-material/ViewTimeline';
import { apiInstance } from '../../services/api';
import { useSessionState } from '../../hooks/useSessionState';
import { HeuresPanel } from './HeuresPanel';
import { PROGRAMME_WORKFLOW, PROGRAMME_LAST_PERIODE_KEY, comparerGroupes } from './def';
import type { GroupeDetail, ReservationDetail } from './def';


// ─── Couleurs par type ────────────────────────────────────────────────────────

const TYPE_COURS_COLORS: Record<string, string> = {
    CM: '#1976d2',
    TD: '#2e7d32',
    TP: '#e65100',
    EXAMEN: '#c62828',
    RATTRAPAGE: '#6a1b9a',
};

const COULEUR_DEFAUT = '#607d8b';
// Bloc fusionné de plusieurs matières : aucune ne peut lui donner sa couleur.
const COULEUR_FUSION = '#455a64';

type ColorMode = 'type' | 'matiere';

// Génère une couleur HSL déterministe à partir d'un entier (matiere_id).
function defaultMatiereColor(id: number): string {
    const hue = (id * 47) % 360;
    return `hsl(${hue}, 55%, 40%)`;
}

function couleurSeance(r: ReservationDetail, colorMode: ColorMode): string {
    if (colorMode === 'matiere') {
        return r.matiere_color ?? (r.matiere_id ? defaultMatiereColor(r.matiere_id) : COULEUR_DEFAUT);
    }
    return TYPE_COURS_COLORS[r.type_cours ?? ''] ?? COULEUR_DEFAUT;
}

// ─── Libellés ─────────────────────────────────────────────────────────────────

// Les matières importées portent presque toutes le nom de la promotion en
// suffixe : « 7.2.01 / UEE ACOUSTIQUE (2A FISE (S7)) ». Sur le planning d'une
// période, ce suffixe ne dit rien et mange la place du bloc. On retire le
// suffixe parenthésé le plus fréquent s'il est porté par la majorité des
// matières — et rien sinon : un « (option) » isolé fait partie du nom.
const SUFFIXE = /\s*\((?:[^()]|\([^()]*\))*\)\s*$/;

function suffixeCommun(reservations: ReservationDetail[]): string {
    const matieres = new Set(reservations.map(r => (r.matiere_name ?? '').trim()));
    const compte = new Map<string, number>();
    for (const m of matieres) {
        const s = SUFFIXE.exec(m)?.[0].trim();
        if (s) compte.set(s, (compte.get(s) ?? 0) + 1);
    }
    let meilleur = '', n = 0;
    for (const [s, c] of compte) if (c > n) { meilleur = s; n = c; }
    return n * 2 >= matieres.size ? meilleur : '';
}

function nomMatiere(r: ReservationDetail, suffixe: string): string {
    const m = (r.matiere_name ?? '').trim();
    return suffixe && m.endsWith(suffixe) ? m.slice(0, -suffixe.length).trim() : m;
}

// Les données importées écrivent « - » pour une salle ou un groupe absent :
// un tiret n'est pas un nom.
function nommes(noms: string[]): string[] {
    return noms.filter(n => n && n.trim() !== '-');
}

function nomsGroupes(r: ReservationDetail): string {
    return nommes(r.groupes.map(g => g.name)).sort(comparerGroupes).join(', ');
}

function nomsSalles(r: ReservationDetail): string {
    return nommes(r.salles.map(s => s.name)).join(', ');
}

function nomsProfs(r: ReservationDetail): string {
    return nommes(r.intervenants.map(i => [i.firstName, i.lastName].filter(Boolean).join(' '))).join(', ');
}

// Le groupe ouvre le titre : dans le planning d'une promotion, c'est lui qui
// distingue deux blocs voisins. Pas de « Cours » de repli quand le type est
// inconnu — un mot identique sur tous les blocs n'est pas une information.
function titreSeance(r: ReservationDetail, suffixe: string): string {
    return [nomsGroupes(r), r.type_cours, nomMatiere(r, suffixe), nomsSalles(r)].filter(Boolean).join(' · ');
}

// ─── Événements ───────────────────────────────────────────────────────────────

// À partir de ce nombre de séances au même horaire, le calendrier ne les peint
// plus côte à côte mais en un seul bloc : la colonne du jour se découperait
// en tranches où plus rien ne se lit (14 UEE en parallèle un jeudi de S7).
// Deux séances côte à côte restent lisibles. Le détail des séances fusionnées
// s'ouvre au clic sur le bloc.
const SEUIL_FUSION = 3;

interface PropsEvenement {
    lignes: string[];
    remarque: string;
    fusion: ReservationDetail[] | null;
}

function evenementSeance(r: ReservationDetail, suffixe: string, colorMode: ColorMode) {
    const color = couleurSeance(r, colorMode);
    const props: PropsEvenement = { lignes: [nomsProfs(r)].filter(Boolean), remarque: r.remarque ?? '', fusion: null };
    return {
        id: String(r.id),
        title: titreSeance(r, suffixe),
        start: r.horaire.Lower,
        end: r.horaire.Upper,
        backgroundColor: color,
        borderColor: color,
        extendedProps: props,
    };
}

// Un bloc pour N séances au même horaire. Une seule matière (16 groupes
// d'anglais) : elle donne le titre et la couleur, la ligne liste les groupes.
// Plusieurs (14 UEE, ou deux TP et un TD) : le bloc dit combien de séances,
// puis une ligne par matière avec ses groupes — les salles sont au clic.
function evenementFusion(seances: ReservationDetail[], suffixe: string, colorMode: ColorMode) {
    const parMatiere = new Map<string, string[]>();
    for (const r of seances) {
        const m = nomMatiere(r, suffixe);
        parMatiere.set(m, [...(parMatiere.get(m) ?? []), ...nommes(r.groupes.map(g => g.name))]);
    }
    const matieres = [...parMatiere.keys()].sort((a, b) => a.localeCompare(b, 'fr', { numeric: true }));
    const groupes = [...parMatiere.values()].flat().sort(comparerGroupes);
    const unique = matieres.length === 1;
    const color = unique ? couleurSeance(seances[0], colorMode) : COULEUR_FUSION;
    const effectif = groupes.length === seances.length ? `${groupes.length} groupes` : `${seances.length} séances`;
    const props: PropsEvenement = {
        lignes: unique
            ? [groupes.join(', ')].filter(Boolean)
            : matieres.map(m => [m, parMatiere.get(m)!.sort(comparerGroupes).join(', ')].filter(Boolean).join(' · ')),
        remarque: '',
        fusion: seances,
    };
    return {
        id: `fusion:${seances[0].horaire.Lower}|${seances[0].horaire.Upper}`,
        title: unique ? `${matieres[0]} · ${effectif}` : `${seances.length} séances en parallèle`,
        start: seances[0].horaire.Lower,
        end: seances[0].horaire.Upper,
        backgroundColor: color,
        borderColor: color,
        extendedProps: props,
    };
}

function construireEvenements(reservations: ReservationDetail[], suffixe: string, colorMode: ColorMode) {
    const parHoraire = new Map<string, ReservationDetail[]>();
    for (const r of reservations) {
        const k = `${r.horaire.Lower}|${r.horaire.Upper}`;
        parHoraire.set(k, [...(parHoraire.get(k) ?? []), r]);
    }
    return [...parHoraire.values()].flatMap(seances =>
        seances.length >= SEUIL_FUSION
            ? [evenementFusion(seances, suffixe, colorMode)]
            : seances.map(r => evenementSeance(r, suffixe, colorMode)),
    );
}

// ─── Composant ────────────────────────────────────────────────────────────────

export function Planning() {
    const { periodeId } = useParams();
    const navigate = useNavigate();

    // Mémorise la période affichée pour reprendre ce planning au retour sur l'écran.
    useEffect(() => {
        if (periodeId) sessionStorage.setItem(PROGRAMME_LAST_PERIODE_KEY, periodeId);
    }, [periodeId]);

    // ── Données ─────────────────────────────────────────────────────────────
    // Mêmes clés que l'écran Groupes : mêmes endpoints, mêmes paramètres — le
    // cache est partagé, l'aller-retour entre les deux vues ne recharge rien.
    const { data: reservations = [] } = useQuery<ReservationDetail[]>({
        queryKey: ['programme.reservations', periodeId],
        queryFn: () => apiInstance.get(`/api/v2/planning/reservation?periode_id=${periodeId}`).then(r => r.data),
        enabled: !!periodeId,
    });
    const { data: groupesData } = useQuery<GroupeDetail[]>({
        queryKey: ['programme.groupes', periodeId],
        queryFn: () => apiInstance.get(`/api/v2/planning/groupes?periode_id=${periodeId}`).then(r => r.data),
        enabled: !!periodeId,
    });
    const groupes = useMemo(
        () => [...(groupesData ?? [])].sort((a, b) => comparerGroupes(a.groupe_name, b.groupe_name)),
        [groupesData],
    );

    // ── Filtre par groupe ────────────────────────────────────────────────────
    // Un identifiant de groupe n'a de sens que dans sa promotion : la valeur
    // mémorisée porte la période, et ne vaut que si c'est encore celle-ci.
    const [groupeMemo, setGroupeMemo] = useSessionState<string>('planning_groupe', '');
    const [memoPeriode, memoGroupe] = groupeMemo.split(':');
    const groupeId = memoPeriode === periodeId && memoGroupe ? Number(memoGroupe) : null;
    const choisirGroupe = (id: number | null) => setGroupeMemo(id == null ? '' : `${periodeId}:${id}`);

    // Le planning d'un groupe : ses séances propres et les amphis de toute la
    // promotion — ce que voit un élève de ce groupe.
    const visibles = useMemo(() => {
        if (groupeId == null) return reservations;
        return reservations.filter(r =>
            r.groupes.length === 0 ? r.promotion_id != null : r.groupes.some(g => g.id === groupeId),
        );
    }, [reservations, groupeId]);

    // ── Persistance légère de l'affichage ────────────────────────────────────
    const DATE_KEY = 'planning_date';
    const initialDate = sessionStorage.getItem(DATE_KEY) ?? undefined;
    const handleDatesSet = useCallback((info: { startStr: string }) => {
        sessionStorage.setItem(DATE_KEY, info.startStr);
    }, []);

    const [colorMode, setColorMode] = useState<ColorMode>(
        () => (sessionStorage.getItem('planning_color_mode') as ColorMode | null) ?? 'type'
    );
    useEffect(() => { sessionStorage.setItem('planning_color_mode', colorMode); }, [colorMode]);

    const [panelOpen, setPanelOpen] = useState(
        () => sessionStorage.getItem('planning_panel_open') !== 'false'
    );
    useEffect(() => { sessionStorage.setItem('planning_panel_open', String(panelOpen)); }, [panelOpen]);

    // ── Événements ───────────────────────────────────────────────────────────
    // Le suffixe se calcule sur toute la période, pas sur le groupe filtré :
    // le libellé d'une matière ne change pas selon le filtre.
    const suffixe = useMemo(() => suffixeCommun(reservations), [reservations]);
    const events = useMemo(() => construireEvenements(visibles, suffixe, colorMode), [visibles, suffixe, colorMode]);

    // Détail d'un bloc fusionné, ancré sur le bloc cliqué.
    const [detail, setDetail] = useState<{ anchor: HTMLElement; seances: ReservationDetail[] } | null>(null);
    const handleEventClick = useCallback((info: EventClickArg) => {
        const { fusion } = info.event.extendedProps as PropsEvenement;
        if (!fusion) return;
        info.jsEvent.preventDefault();
        const seances = [...fusion].sort((a, b) => comparerGroupes(nomsGroupes(a), nomsGroupes(b)));
        setDetail({ anchor: info.el, seances });
    }, []);

    const renderEvent = useCallback((info: EventContentArg) => {
        const { lignes, remarque } = info.event.extendedProps as PropsEvenement;
        return (
            <Box sx={{
                fontSize: '0.75rem', height: '100%', overflow: 'hidden', px: 0.5, lineHeight: 1.3,
                // Fondu sur la dernière ligne coupée : elle se lit comme « il y a la suite », pas comme un bug.
                maskImage: 'linear-gradient(#000 calc(100% - 12px), transparent)',
            }}>
                <strong>{info.event.title}</strong>
                {lignes.map((l, i) => <div key={i}>{l}</div>)}
                {remarque && <div style={{ fontStyle: 'italic' }}>{remarque}</div>}
            </Box>
        );
    }, []);

    // ── Rendu ────────────────────────────────────────────────────────────────
    return (
        <Box sx={{ height: '100%', display: 'flex', overflow: 'hidden' }}>

            {/* Calendrier */}
            <Box sx={{ flex: 1, minWidth: 0, p: 2, display: 'flex', flexDirection: 'column' }}>
                <Stack direction="row" justifyContent="flex-end" sx={{ mb: 1 }}>
                    <FormControl size="small" sx={{ minWidth: 240 }}>
                        <InputLabel id="planning-groupe-label">Groupe</InputLabel>
                        <Select<number | ''>
                            labelId="planning-groupe-label"
                            label="Groupe"
                            value={groupeId ?? ''}
                            onChange={e => choisirGroupe(e.target.value === '' ? null : Number(e.target.value))}
                        >
                            <MenuItem value="">Tous les groupes</MenuItem>
                            {groupes.map(g => (
                                <MenuItem key={g.groupe_id} value={g.groupe_id}>{g.groupe_name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                </Stack>
                {/* Le texte d'un bloc s'arrête à la hauteur de la séance : le
                    reste est dans la bulle de détail, pas sur le bloc voisin. */}
                <Box sx={{ flex: 1, minHeight: 0, '& .fc-timegrid-event': { overflow: 'hidden' }, '& .fc-event.fusion': { cursor: 'pointer' } }}>
                    <FullCalendar
                        plugins={[timeGridPlugin, dayGridPlugin]}
                        initialView="timeGridWeek"
                        initialDate={initialDate}
                        datesSet={handleDatesSet}
                        locale={frLocale}
                        headerToolbar={{
                            left: 'prev,next today',
                            center: 'title',
                            right: 'dayGridMonth,timeGridWeek,timeGridDay',
                        }}
                        events={events}
                        eventClassNames={info => (info.event.extendedProps as PropsEvenement).fusion ? ['fusion'] : []}
                        eventClick={handleEventClick}
                        weekends={false}
                        allDaySlot={false}
                        slotMinTime="07:00:00"
                        slotMaxTime="21:00:00"
                        height="100%"
                        selectable={false}
                        editable={false}
                        eventContent={renderEvent}
                    />
                </Box>
            </Box>

            {/* Détail d'un bloc fusionné */}
            <Popover
                open={detail != null}
                anchorEl={detail?.anchor}
                onClose={() => setDetail(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
            >
                {detail && (
                    <Box sx={{ p: 2, maxWidth: 560, maxHeight: 440, overflow: 'auto' }}>
                        <Typography variant="subtitle2" gutterBottom>
                            {dayjs(detail.seances[0].horaire.Lower).toDate().toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
                            {' · '}{dayjs(detail.seances[0].horaire.Lower).format('HH:mm')}–{dayjs(detail.seances[0].horaire.Upper).format('HH:mm')}
                            {' · '}{detail.seances.length} séances en parallèle
                        </Typography>
                        <Stack divider={<Divider flexItem />} spacing={0.75}>
                            {detail.seances.map(r => {
                                const salles = nomsSalles(r);
                                const profs = nomsProfs(r);
                                return (
                                    <Box key={r.id}>
                                        <Typography variant="body2">
                                            <strong>{nomsGroupes(r) || 'Toute la promotion'}</strong>
                                            {' · '}{nomMatiere(r, suffixe)}
                                            {salles && ` · ${salles}`}
                                        </Typography>
                                        {profs && (
                                            <Typography variant="caption" color="text.secondary" display="block">{profs}</Typography>
                                        )}
                                        {r.remarque && (
                                            <Typography variant="caption" fontStyle="italic" display="block">{r.remarque}</Typography>
                                        )}
                                    </Box>
                                );
                            })}
                        </Stack>
                    </Box>
                )}
            </Popover>

            {/* Boutons toggle */}
            <Box sx={{ display: 'flex', flexDirection: 'column', alignSelf: 'flex-start', mt: 2, mr: panelOpen ? 0 : 1, gap: 0.5 }}>
                <Tooltip title="Changer de promotion / période" placement="left">
                    <IconButton size="small" onClick={() => navigate(`/${PROGRAMME_WORKFLOW}/select`)}>
                        <TuneIcon />
                    </IconButton>
                </Tooltip>
                <Tooltip title="Vue par groupes" placement="left">
                    <IconButton size="small" onClick={() => navigate(`/${PROGRAMME_WORKFLOW}/${periodeId}/groupes`)}>
                        <ViewTimelineIcon />
                    </IconButton>
                </Tooltip>
                <Tooltip title={colorMode === 'type' ? 'Couleur par matière' : 'Couleur par type de cours'} placement="left">
                    <IconButton
                        size="small"
                        onClick={() => setColorMode(m => m === 'type' ? 'matiere' : 'type')}
                        color={colorMode === 'matiere' ? 'primary' : 'default'}
                    >
                        <PaletteIcon />
                    </IconButton>
                </Tooltip>
                <Tooltip title={panelOpen ? 'Masquer les heures' : 'Afficher les heures'} placement="left">
                    <IconButton size="small" onClick={() => setPanelOpen(p => !p)}>
                        <MenuOpenIcon sx={{ transform: panelOpen ? 'none' : 'scaleX(-1)' }} />
                    </IconButton>
                </Tooltip>
            </Box>

            {/* Panel heures consommées */}
            {panelOpen && <HeuresPanel periodeId={periodeId ?? ''} />}
        </Box>
    );
}
