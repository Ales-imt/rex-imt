import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import type { NpsDistribution } from '../Evaluation';

interface Props {
    distribution: NpsDistribution;
}

export function NpsBar({ distribution }: Props) {
    const theme = useTheme();
    const { pct_detracteurs, pct_passifs, pct_promoteurs } = distribution;

    return (
        <Box sx={{
            bgcolor: theme.palette.background.paper,
            border: '0.5px solid',
            borderColor: theme.palette.divider,
            borderRadius: '12px',
            p: 2.5,
        }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1.5 }}>
                Distribution NPS
            </Typography>
            <Box sx={{ display: 'flex', borderRadius: 2, overflow: 'hidden', height: 24 }}>
                {pct_detracteurs > 0 && (
                    <Box sx={{ width: `${pct_detracteurs}%`, bgcolor: '#ef4444', transition: 'width 0.4s ease' }} />
                )}
                {pct_passifs > 0 && (
                    <Box sx={{ width: `${pct_passifs}%`, bgcolor: '#f59e0b', transition: 'width 0.4s ease' }} />
                )}
                {pct_promoteurs > 0 && (
                    <Box sx={{ width: `${pct_promoteurs}%`, bgcolor: '#10b981', transition: 'width 0.4s ease' }} />
                )}
                {pct_detracteurs === 0 && pct_passifs === 0 && pct_promoteurs === 0 && (
                    <Box sx={{ width: '100%', bgcolor: theme.palette.action.hover }} />
                )}
            </Box>
            <Box sx={{ display: 'flex', gap: 3, mt: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: '#ef4444' }} />
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        Détracteurs {pct_detracteurs.toFixed(0)}%
                    </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: '#f59e0b' }} />
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        Passifs {pct_passifs.toFixed(0)}%
                    </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: '#10b981' }} />
                    <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        Promoteurs {pct_promoteurs.toFixed(0)}%
                    </Typography>
                </Box>
            </Box>
        </Box>
    );
}
