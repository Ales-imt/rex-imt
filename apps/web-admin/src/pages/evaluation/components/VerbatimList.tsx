import { useState } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import { useTheme } from '@mui/material/styles';
import { useVerbatims } from '../useEvaluations';

const DIMENSION_COLORS: Record<string, { bg: string; color: string; border: string }> = {
    SUPPORTS: { bg: '#eff6ff', color: '#1d4ed8', border: '#bfdbfe' },
    AMBIANCE: { bg: '#fdf4ff', color: '#7e22ce', border: '#e9d5ff' },
    NPS:      { bg: '#fff7ed', color: '#c2410c', border: '#fed7aa' },
};

interface Props {
    matiereId: number;
}

export function VerbatimList({ matiereId }: Props) {
    const theme = useTheme();
    const [dimension, setDimension] = useState('');
    const [page, setPage] = useState(1);

    const { data, loading } = useVerbatims(matiereId, dimension || undefined, page);
    const items = data?.data ?? [];
    const hasNext = items.length === (data?.limit ?? 10);

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <FormControl size="small" sx={{ maxWidth: 220 }}>
                <InputLabel>Dimension</InputLabel>
                <Select
                    value={dimension}
                    label="Dimension"
                    onChange={e => { setDimension(e.target.value); setPage(1); }}
                >
                    <MenuItem value="">Toutes</MenuItem>
                    <MenuItem value="SUPPORTS">Supports</MenuItem>
                    <MenuItem value="AMBIANCE">Ambiance</MenuItem>
                    <MenuItem value="NPS">NPS</MenuItem>
                </Select>
            </FormControl>

            {loading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                    <CircularProgress size={28} />
                </Box>
            )}

            {!loading && items.length === 0 && (
                <Typography variant="body2" sx={{ color: '#94a3b8', textAlign: 'center', py: 4 }}>
                    Aucun verbatim
                </Typography>
            )}

            {!loading && items.map(v => {
                const dim = DIMENSION_COLORS[v.dimension] ?? { bg: theme.palette.action.hover, color: 'text.primary', border: 'divider' };
                const date = new Date(v.created_at).toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric' });
                return (
                    <Box key={v.id} sx={{
                        bgcolor: theme.palette.background.paper,
                        border: '0.5px solid',
                        borderColor: theme.palette.divider,
                        borderRadius: '12px',
                        p: 2,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 1,
                    }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Box sx={{
                                bgcolor: dim.bg,
                                color: dim.color,
                                border: `0.5px solid ${dim.border}`,
                                borderRadius: '6px',
                                px: 1,
                                py: 0.25,
                            }}>
                                <Typography variant="caption" fontWeight={600}>{v.dimension}</Typography>
                            </Box>
                            <Typography variant="caption" sx={{ color: '#94a3b8', ml: 'auto' }}>{date}</Typography>
                        </Box>
                        <Typography variant="body2" sx={{ color: 'text.primary', lineHeight: 1.6 }}>
                            {v.texte}
                        </Typography>
                    </Box>
                );
            })}

            {(page > 1 || hasNext) && (
                <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center' }}>
                    <Button
                        variant="outlined"
                        size="small"
                        disabled={page === 1}
                        onClick={() => setPage(p => p - 1)}
                    >
                        Précédent
                    </Button>
                    <Typography variant="body2" sx={{ alignSelf: 'center', color: 'text.secondary' }}>
                        Page {page}
                    </Typography>
                    <Button
                        variant="outlined"
                        size="small"
                        disabled={!hasNext}
                        onClick={() => setPage(p => p + 1)}
                    >
                        Suivant
                    </Button>
                </Box>
            )}
        </Box>
    );
}
