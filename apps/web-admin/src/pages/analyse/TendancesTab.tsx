import { useMemo } from 'react';
import { Box, Paper, Typography } from '@mui/material';
import type { Feedback } from '../feedback/Feedback';
import { classifiedFeedbacks, HorizontalBar, KpiCard, SENTIMENT_COLOR, SENTIMENT_LABEL } from './shared';

export function TendancesTab({ data }: { data: Feedback[] }) {
    const classified = useMemo(() => classifiedFeedbacks(data), [data]);
    const pending = data.length - classified.length;
    const pendingPct = data.length > 0 ? `${Math.round((pending / data.length) * 100)} %` : '—';

    const byCategorie = useMemo(() => {
        const map = new Map<string, number>();
        classified.forEach(f => {
            const key = f.categorie ?? 'Non classifié';
            map.set(key, (map.get(key) ?? 0) + 1);
        });
        return Array.from(map.entries()).sort((a, b) => b[1] - a[1]);
    }, [classified]);

    const bySentiment = useMemo(() => {
        const map = new Map<string, number>();
        classified.forEach(f => {
            const key = f.sentiment ?? 'inconnu';
            map.set(key, (map.get(key) ?? 0) + 1);
        });
        return Array.from(map.entries()).sort((a, b) => b[1] - a[1]);
    }, [classified]);

    return (
        <Box>
            <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
                <KpiCard label="Total feedbacks reçus" value={data.length} color="#1976d2" />
                <KpiCard label="En attente d'analyse" value={pending} color="#9e9e9e" />
                <KpiCard label="Non encore traités" value={pendingPct} color="#f57c00" />
            </Box>

            <Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
                <Paper elevation={1} sx={{ p: 2.5, flex: 1, minWidth: 260 }}>
                    <Typography variant="subtitle1" fontWeight={600} mb={2}>Répartition par catégorie</Typography>
                    {byCategorie.length === 0
                        ? <Typography variant="body2" color="text.secondary">Aucune donnée</Typography>
                        : byCategorie.map(([label, count]) => (
                            <HorizontalBar key={label} label={label} count={count} total={classified.length} />
                        ))
                    }
                </Paper>

                <Paper elevation={1} sx={{ p: 2.5, flex: 1, minWidth: 260 }}>
                    <Typography variant="subtitle1" fontWeight={600} mb={2}>Répartition par sentiment</Typography>
                    {bySentiment.length === 0
                        ? <Typography variant="body2" color="text.secondary">Aucune donnée</Typography>
                        : bySentiment.map(([key, count]) => (
                            <HorizontalBar
                                key={key}
                                label={SENTIMENT_LABEL[key] ?? key}
                                count={count}
                                total={classified.length}
                                color={SENTIMENT_COLOR[key]}
                            />
                        ))
                    }
                </Paper>
            </Box>
        </Box>
    );
}
