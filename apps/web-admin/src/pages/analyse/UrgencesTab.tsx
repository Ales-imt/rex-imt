import { useMemo, useState } from 'react';
import {
    Alert,
    Box,
    Card,
    CardContent,
    Chip,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
    Typography,
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import type { Feedback } from '../feedback/Feedback';
import { classifiedFeedbacks, KpiCard, relativeTime, urgenceColor } from './shared';

function FeedbackCard({ feedback: f }: { feedback: Feedback }) {
    const color = urgenceColor(f.urgence);
    const excerpt = f.content?.length > 120 ? `${f.content.slice(0, 120)}…` : f.content;

    return (
        <Card elevation={1} sx={{ borderLeft: `5px solid ${color}`, borderRadius: 1 }}>
            <CardContent sx={{ pb: '12px !important' }}>
                {f.resume && (
                    <Typography variant="body1" fontWeight={500} mb={0.5}>{f.resume}</Typography>
                )}
                <Tooltip title={f.content} placement="top" arrow>
                    <Typography variant="body2" color="text.secondary" mb={1} sx={{ cursor: 'default' }}>{excerpt}</Typography>
                </Tooltip>
                <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', alignItems: 'center' }}>
                    {f.categorie && <Chip label={f.categorie} size="small" variant="outlined" />}
                    {f.promotion && <Chip label={f.promotion} size="small" variant="outlined" color="primary" />}
                    {f.urgence != null && (
                        <Chip
                            label={`Urgence ${f.urgence}`}
                            size="small"
                            sx={{ bgcolor: color, color: f.urgence <= 1 ? '#555' : '#fff', fontWeight: 600 }}
                        />
                    )}
                    <Typography variant="caption" color="text.disabled" ml="auto">
                        {relativeTime(f.created_at)}
                    </Typography>
                </Box>
            </CardContent>
        </Card>
    );
}

export function UrgencesTab({ data }: { data: Feedback[] }) {
    const [activeCategory, setActiveCategory] = useState<string | null>(null);

    const classified = useMemo(() => classifiedFeedbacks(data), [data]);
    const critical = useMemo(() => classified.filter(f => f.urgence === 5), [classified]);
    const high = useMemo(() => classified.filter(f => (f.urgence ?? 0) >= 3 && (f.urgence ?? 0) <= 4), [classified]);
    const positifPct = useMemo(() => {
        if (classified.length === 0) return '—';
        const n = classified.filter(f => f.sentiment === 'positif').length;
        return `${Math.round((n / classified.length) * 100)} %`;
    }, [classified]);

    const categories = useMemo(() => {
        const set = new Set(classified.map(f => f.categorie).filter(Boolean) as string[]);
        return Array.from(set).sort();
    }, [classified]);

    const displayed = useMemo(() => {
        const base = activeCategory
            ? classified.filter(f => f.categorie === activeCategory)
            : classified;
        return [...base].sort((a, b) => (b.urgence ?? 0) - (a.urgence ?? 0)).slice(0, 50);
    }, [classified, activeCategory]);

    return (
        <Box>
            {critical.length > 0 && (
                <Alert severity="error" icon={<WarningAmberIcon />} sx={{ mb: 2, fontWeight: 'bold' }}>
                    {critical.length} feedback{critical.length > 1 ? 's' : ''} nécessite{critical.length > 1 ? 'nt' : ''} une attention immédiate
                </Alert>
            )}

            <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
                <KpiCard label="Urgences critiques" value={critical.length} color="#d32f2f" />
                <KpiCard label="Urgences élevées (3–4)" value={high.length} color="#f57c00" />
                <KpiCard label="Feedbacks positifs" value={positifPct} color="#2e7d32" />
            </Box>

            {categories.length > 0 && (
                <Box sx={{ mb: 2, display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                    <Typography variant="body2" color="text.secondary" mr={1}>Filtrer :</Typography>
                    <ToggleButtonGroup
                        value={activeCategory}
                        exclusive
                        onChange={(_, v) => setActiveCategory(v)}
                        size="small"
                        sx={{ flexWrap: 'wrap', gap: 0.5 }}
                    >
                        {categories.map(cat => (
                            <ToggleButton key={cat} value={cat} sx={{ textTransform: 'none', fontSize: '0.75rem' }}>
                                {cat}
                            </ToggleButton>
                        ))}
                    </ToggleButtonGroup>
                </Box>
            )}

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                {displayed.map(f => <FeedbackCard key={f.id} feedback={f} />)}
                {displayed.length === 0 && (
                    <Typography color="text.secondary" textAlign="center" py={4}>
                        Aucun feedback classifié pour le moment.
                    </Typography>
                )}
            </Box>
        </Box>
    );
}
