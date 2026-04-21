import { Box, LinearProgress, Paper, Typography } from '@mui/material';
import type { Feedback } from '../feedback/Feedback';

export const URGENCE_COLOR: Record<number, string> = {
    5: '#d32f2f',
    4: '#f57c00',
    3: '#1976d2',
    2: '#9e9e9e',
    1: '#e0e0e0',
};

export const SENTIMENT_LABEL: Record<string, string> = {
    positif: 'Positif',
    negatif: 'Négatif',
    neutre: 'Neutre',
    mitige: 'Mitigé',
};

export const SENTIMENT_COLOR: Record<string, string> = {
    positif: '#2e7d32',
    negatif: '#c62828',
    neutre: '#1565c0',
    mitige: '#e65100',
};

export function urgenceColor(u: number | null | undefined): string {
    return u != null ? (URGENCE_COLOR[u] ?? '#e0e0e0') : '#e0e0e0';
}

export function relativeTime(dateStr: string | null | undefined): string {
    if (!dateStr) return '';
    const diff = Date.now() - new Date(dateStr).getTime();
    const m = Math.floor(diff / 60000);
    const h = Math.floor(m / 60);
    const d = Math.floor(h / 24);
    if (d > 0) return `il y a ${d} jour${d > 1 ? 's' : ''}`;
    if (h > 0) return `il y a ${h} heure${h > 1 ? 's' : ''}`;
    if (m > 0) return `il y a ${m} minute${m > 1 ? 's' : ''}`;
    return "à l'instant";
}

export function classifiedFeedbacks(data: Feedback[]): Feedback[] {
    return data.filter(f => f.urgence != null);
}

export function KpiCard({ label, value, color = '#1976d2' }: { label: string; value: string | number; color?: string }) {
    return (
        <Paper elevation={1} sx={{ p: 2.5, textAlign: 'center', borderTop: `4px solid ${color}`, flex: 1, minWidth: 130 }}>
            <Typography variant="h4" fontWeight="bold" color={color}>{value}</Typography>
            <Typography variant="body2" color="text.secondary" mt={0.5}>{label}</Typography>
        </Paper>
    );
}

export function HorizontalBar({ label, count, total, color }: { label: string; count: number; total: number; color?: string }) {
    const pct = total > 0 ? Math.round((count / total) * 100) : 0;
    return (
        <Box sx={{ mb: 1.5 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                <Typography variant="body2">{label}</Typography>
                <Typography variant="body2" color="text.secondary">{count} ({pct} %)</Typography>
            </Box>
            <LinearProgress
                variant="determinate"
                value={pct}
                sx={{
                    height: 10,
                    borderRadius: 1,
                    bgcolor: '#f0f0f0',
                    '& .MuiLinearProgress-bar': { bgcolor: color ?? '#1976d2', borderRadius: 1 },
                }}
            />
        </Box>
    );
}
