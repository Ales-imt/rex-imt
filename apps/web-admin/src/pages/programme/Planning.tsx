import FullCalendar from '@fullcalendar/react';
import timeGridPlugin from '@fullcalendar/timegrid';
import dayGridPlugin from '@fullcalendar/daygrid';
import frLocale from '@fullcalendar/core/locales/fr';
import { useQuery } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router';
import { Box, IconButton, Tooltip } from '@mui/material';
import { useState, useEffect, useCallback } from 'react';
import MenuOpenIcon from '@mui/icons-material/MenuOpen';
import PaletteIcon from '@mui/icons-material/Palette';
import TuneIcon from '@mui/icons-material/Tune';
import { apiInstance } from '../../services/api';
import { HeuresPanel } from './HeuresPanel';
import { PROGRAMME_WORKFLOW, PROGRAMME_LAST_PERIODE_KEY } from './def';
import type { ReservationDetail } from './def';


// ─── Couleurs par type ────────────────────────────────────────────────────────

const TYPE_COURS_COLORS: Record<string, string> = {
    CM: '#1976d2',
    TD: '#2e7d32',
    TP: '#e65100',
    EXAMEN: '#c62828',
    RATTRAPAGE: '#6a1b9a',
};

type ColorMode = 'type' | 'matiere';

// Génère une couleur HSL déterministe à partir d'un entier (matiere_id).
function defaultMatiereColor(id: number): string {
    const hue = (id * 47) % 360;
    return `hsl(${hue}, 55%, 40%)`;
}

function toEvent(r: ReservationDetail, colorMode: ColorMode) {
    const typeLabel = r.type_cours ?? r.description ?? 'Cours';
    const matiereLabel = r.matiere_name ?? '';
    const sallesLabel = r.salles.map(s => s.name).join(', ');
    const intervenantsLabel = r.intervenants
        .map(i => [i.firstName, i.lastName].filter(Boolean).join(' '))
        .join(', ');

    const color = colorMode === 'matiere'
        ? (r.matiere_color ?? (r.matiere_id ? defaultMatiereColor(r.matiere_id) : '#607d8b'))
        : (TYPE_COURS_COLORS[r.type_cours ?? ''] ?? '#607d8b');

    return {
        id: String(r.id),
        title: [typeLabel, matiereLabel, sallesLabel].filter(Boolean).join(' — '),
        start: r.horaire.Lower,
        end: r.horaire.Upper,
        backgroundColor: color,
        borderColor: color,
        extendedProps: { intervenantsLabel, remarque: r.remarque ?? '' },
    };
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
    const { data: reservations = [] } = useQuery<ReservationDetail[]>({
        queryKey: ['reservations', periodeId],
        queryFn: () => apiInstance.get(`/api/v2/planning/reservation?periode_id=${periodeId}`).then(r => r.data),
        enabled: !!periodeId,
    });

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

    // ── Rendu ────────────────────────────────────────────────────────────────
    return (
        <Box sx={{ height: '100%', display: 'flex', overflow: 'hidden' }}>

            {/* Calendrier */}
            <Box sx={{ flex: 1, minWidth: 0, p: 2, display: 'flex', flexDirection: 'column' }}>
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
                    events={reservations.map(r => toEvent(r, colorMode))}
                    weekends={false}
                    allDaySlot={false}
                    slotMinTime="07:00:00"
                    slotMaxTime="21:00:00"
                    height="100%"
                    selectable={false}
                    editable={false}
                    eventContent={(info) => (
                        <Box sx={{ fontSize: '0.75rem', overflow: 'hidden', px: 0.5, lineHeight: 1.3 }}>
                            <strong>{info.event.title}</strong>
                            {info.event.extendedProps.intervenantsLabel && (
                                <div>{info.event.extendedProps.intervenantsLabel}</div>
                            )}
                            {info.event.extendedProps.remarque && (
                                <div style={{ fontStyle: 'italic' }}>{info.event.extendedProps.remarque}</div>
                            )}
                        </Box>
                    )}
                />
            </Box>

            {/* Boutons toggle */}
            <Box sx={{ display: 'flex', flexDirection: 'column', alignSelf: 'flex-start', mt: 2, mr: panelOpen ? 0 : 1, gap: 0.5 }}>
                <Tooltip title="Changer de promotion / période" placement="left">
                    <IconButton size="small" onClick={() => navigate(`/${PROGRAMME_WORKFLOW}/select`)}>
                        <TuneIcon />
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
