import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import type { EvalStats } from '../Evaluation';

interface Props {
    stats: EvalStats;
}

function Card({ label, value, sub, alert }: { label: string; value: string; sub?: string; alert?: boolean }) {
    const theme = useTheme();
    return (
        <Box sx={{
            flex: 1,
            minWidth: 0,
            bgcolor: theme.palette.background.paper,
            border: '0.5px solid',
            borderColor: theme.palette.divider,
            borderRadius: '12px',
            p: 2,
            display: 'flex',
            flexDirection: 'column',
            gap: 0.5,
        }}>
            <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 500, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                {label}
            </Typography>
            <Typography variant="h5" fontWeight={700} sx={{ color: alert ? '#ef4444' : 'text.primary' }}>
                {value}
            </Typography>
            {sub && (
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>{sub}</Typography>
            )}
        </Box>
    );
}

export function MetricCards({ stats }: Props) {
    const scoreGlobal = [stats.score_peda, stats.score_contenu, stats.score_supports, stats.score_ambiance]
        .filter((v): v is number => v != null);
    const avgGlobal = scoreGlobal.length > 0
        ? (scoreGlobal.reduce((a, b) => a + b, 0) / scoreGlobal.length).toFixed(1)
        : '—';

    const nps = stats.nps_moyen;

    return (
        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <Card
                label="Répondants"
                value={String(stats.nb_repondants)}
                sub="étudiants ayant soumis"
            />
            <Card
                label="Score global"
                value={`${avgGlobal} / 5`}
                sub="péda · contenu · supports · ambiance"
            />
            <Card
                label="NPS moyen"
                value={nps != null ? `${nps.toFixed(1)} / 10` : '—'}
                sub={nps != null ? (nps >= 7 ? 'Bon' : nps >= 5 ? 'Moyen' : 'Faible') : undefined}
            />
            <Box sx={{
                flex: 1,
                minWidth: 0,
                bgcolor: stats.pct_charge_lourde > 40 ? '#fef2f2' : 'background.paper',
                border: '0.5px solid',
                borderColor: stats.pct_charge_lourde > 40 ? '#fecaca' : 'divider',
                borderRadius: '12px',
                p: 2,
                display: 'flex',
                flexDirection: 'column',
                gap: 0.5,
            }}>
                <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 500, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                    Charge lourde
                </Typography>
                <Typography variant="h5" fontWeight={700} sx={{ color: stats.pct_charge_lourde > 40 ? '#ef4444' : 'text.primary' }}>
                    {stats.pct_charge_lourde.toFixed(0)}%
                </Typography>
                {stats.pct_charge_lourde > 40 && (
                    <Typography variant="caption" sx={{ color: '#ef4444' }}>Charge excessive signalée</Typography>
                )}
            </Box>
            {/* invisible spacer hack for consistent 4-col wrapping */}
            <Box sx={{ flex: 1, minWidth: 0, visibility: 'hidden', p: 2 }} />
        </Box>
    );
}
