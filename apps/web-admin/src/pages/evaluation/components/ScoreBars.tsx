import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import type { EvalStats } from '../Evaluation';

interface Props {
    stats: EvalStats;
}

function barColor(v: number): string {
    if (v >= 4) return '#10b981';
    if (v >= 3) return '#f59e0b';
    return '#ef4444';
}

function ScoreBar({ label, value }: { label: string; value: number | null }) {
    const theme = useTheme();
    const pct = value != null ? (value / 5) * 100 : 0;
    const color = value != null ? barColor(value) : '#cbd5e1';

    return (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Typography variant="body2" sx={{ width: 130, flexShrink: 0, color: 'text.secondary' }}>
                {label}
            </Typography>
            <Box sx={{ flex: 1, bgcolor: theme.palette.action.hover, borderRadius: 4, height: 10, overflow: 'hidden' }}>
                <Box sx={{ width: `${pct}%`, bgcolor: color, height: '100%', borderRadius: 4, transition: 'width 0.4s ease' }} />
            </Box>
            <Typography variant="body2" fontWeight={600} sx={{ width: 36, textAlign: 'right', color: value != null ? color : '#94a3b8' }}>
                {value != null ? value.toFixed(1) : '—'}
            </Typography>
        </Box>
    );
}

export function ScoreBars({ stats }: Props) {
    const theme = useTheme();
    return (
        <Box sx={{
            bgcolor: theme.palette.background.paper,
            border: '0.5px solid',
            borderColor: theme.palette.divider,
            borderRadius: '12px',
            p: 2.5,
            display: 'flex',
            flexDirection: 'column',
            gap: 1.5,
        }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 0.5 }}>
                Scores par dimension
            </Typography>
            <ScoreBar label="Pédagogie" value={stats.score_peda} />
            <ScoreBar label="Contenu" value={stats.score_contenu} />
            <ScoreBar label="Supports" value={stats.score_supports} />
            <ScoreBar label="Ambiance" value={stats.score_ambiance} />
            <ScoreBar label="Difficulté" value={stats.score_difficulte} />
        </Box>
    );
}
