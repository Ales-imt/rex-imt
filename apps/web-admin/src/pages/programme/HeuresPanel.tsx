import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Box, Typography, Divider, ToggleButton, ToggleButtonGroup } from '@mui/material';
import { apiInstance } from '../../services/api';

interface HeuresItem {
    id:                number;
    label:             string;
    heures_consommees: number;
    children?:         HeuresItem[];
}

interface HeuresBreakdown {
    matiere: HeuresItem[];
    groupe:  HeuresItem[];
    prof:    HeuresItem[];
}

type Dimension = 'matiere' | 'groupe' | 'prof';

const DIM_STORAGE_KEY = 'programme.heuresDim';
const DIM_LABELS: Record<Dimension, string> = {
    matiere: 'Matière',
    groupe:  'Groupe',
    prof:    'Prof',
};

interface Props {
    periodeId: string;
}

export function HeuresPanel({ periodeId }: Props) {
    const { data } = useQuery<HeuresBreakdown>({
        queryKey: ['heures', periodeId],
        queryFn:  () => apiInstance.get(`/api/v2/planning/heures?periode_id=${periodeId}`).then(r => r.data),
        enabled:  !!periodeId,
    });

    const [dim, setDim] = useState<Dimension>(
        () => (sessionStorage.getItem(DIM_STORAGE_KEY) as Dimension | null) ?? 'matiere'
    );
    const changeDim = (d: Dimension | null) => {
        if (!d) return;
        setDim(d);
        sessionStorage.setItem(DIM_STORAGE_KEY, d);
    };

    const items = data?.[dim] ?? [];
    const total = items.reduce((s, m) => s + m.heures_consommees, 0);

    return (
        <Box sx={{
            width:      260,
            flexShrink: 0,
            overflowY:  'auto',
            borderLeft: 1,
            borderColor:'divider',
            p:          1.5,
            display:    'flex',
            flexDirection: 'column',
            gap:        1.5,
        }}>
            {/* Total */}
            <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="subtitle2" fontWeight="bold">Total consommé</Typography>
                <Typography variant="subtitle2" color="text.secondary">
                    {total.toFixed(1)}h
                </Typography>
            </Box>

            {/* Bascule de dimension */}
            <ToggleButtonGroup
                size="small"
                exclusive
                fullWidth
                value={dim}
                onChange={(_, d) => changeDim(d)}
            >
                {(Object.keys(DIM_LABELS) as Dimension[]).map(d => (
                    <ToggleButton key={d} value={d} sx={{ textTransform: 'none', py: 0.25 }}>
                        {DIM_LABELS[d]}
                    </ToggleButton>
                ))}
            </ToggleButtonGroup>

            <Divider />

            {/* Répartition selon la dimension choisie */}
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                {items.map(m => (
                    <Box key={`${dim}-${m.id}-${m.label}`}>
                        {/* Ligne principale (matière / groupe / prof) */}
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1 }}>
                            <Typography variant="caption" noWrap sx={{ maxWidth: 170, fontWeight: m.children ? 700 : 500 }}>
                                {m.label}
                            </Typography>
                            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: m.children ? 700 : 400 }}>
                                {m.heures_consommees.toFixed(1)}h
                            </Typography>
                        </Box>
                        {/* Détail par matière (dimension groupe) */}
                        {m.children?.map(c => (
                            <Box key={c.id} sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, pl: 1.5, mt: 0.25 }}>
                                <Typography variant="caption" noWrap sx={{ maxWidth: 160, color: 'text.secondary' }}>
                                    {c.label}
                                </Typography>
                                <Typography variant="caption" color="text.disabled">
                                    {c.heures_consommees.toFixed(1)}h
                                </Typography>
                            </Box>
                        ))}
                    </Box>
                ))}
                {items.length === 0 && (
                    <Typography variant="caption" color="text.disabled">
                        Aucune séance planifiée.
                    </Typography>
                )}
            </Box>
        </Box>
    );
}
