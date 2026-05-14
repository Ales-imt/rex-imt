import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import type { ChipStat } from '../Evaluation';

interface Props {
    chips: ChipStat[];
}

export function ChipSignals({ chips }: Props) {
    const theme = useTheme();
    const positifs = chips.filter(c => c.polarite === 'POSITIF').slice(0, 6);
    const negatifs = chips.filter(c => c.polarite === 'NEGATIF').slice(0, 6);

    return (
        <Box sx={{
            bgcolor: theme.palette.background.paper,
            border: '0.5px solid',
            borderColor: theme.palette.divider,
            borderRadius: '12px',
            p: 2.5,
        }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1.5 }}>
                Signaux qualitatifs
            </Typography>
            <Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
                <Box sx={{ flex: 1, minWidth: 200 }}>
                    <Typography variant="caption" sx={{ color: '#10b981', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                        Points forts
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
                        {positifs.length === 0 && (
                            <Typography variant="caption" sx={{ color: '#94a3b8' }}>Aucun</Typography>
                        )}
                        {positifs.map(c => (
                            <Box key={c.chip_id} sx={{
                                bgcolor: '#f0fdf4',
                                color: '#15803d',
                                border: '0.5px solid #86efac',
                                borderRadius: '6px',
                                px: 1.5,
                                py: 0.5,
                                display: 'flex',
                                alignItems: 'center',
                                gap: 0.5,
                            }}>
                                <Typography variant="caption" fontWeight={500}>{c.libelle}</Typography>
                                <Typography variant="caption" sx={{ opacity: 0.7 }}>{c.pct.toFixed(0)}%</Typography>
                            </Box>
                        ))}
                    </Box>
                </Box>
                <Box sx={{ flex: 1, minWidth: 200 }}>
                    <Typography variant="caption" sx={{ color: '#ef4444', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                        Points faibles
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
                        {negatifs.length === 0 && (
                            <Typography variant="caption" sx={{ color: '#94a3b8' }}>Aucun</Typography>
                        )}
                        {negatifs.map(c => (
                            <Box key={c.chip_id} sx={{
                                bgcolor: '#fef2f2',
                                color: '#b91c1c',
                                border: '0.5px solid #fca5a5',
                                borderRadius: '6px',
                                px: 1.5,
                                py: 0.5,
                                display: 'flex',
                                alignItems: 'center',
                                gap: 0.5,
                            }}>
                                <Typography variant="caption" fontWeight={500}>{c.libelle}</Typography>
                                <Typography variant="caption" sx={{ opacity: 0.7 }}>{c.pct.toFixed(0)}%</Typography>
                            </Box>
                        ))}
                    </Box>
                </Box>
            </Box>
        </Box>
    );
}
