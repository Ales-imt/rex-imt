import { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Accordion from '@mui/material/Accordion';
import AccordionSummary from '@mui/material/AccordionSummary';
import AccordionDetails from '@mui/material/AccordionDetails';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CircularProgress from '@mui/material/CircularProgress';
import { useTheme } from '@mui/material/styles';
import { useAnneePromo } from '../../hooks/useCurriculum';
import { AnneePromoSelect } from '../../components/AnneePromoSelect';
import type { SelectedMatiere } from './Evaluation';

const DOT_COLOR: Record<string, string> = {
    OK:   '#10b981',
    WARN: '#f59e0b',
    NONE: '#cbd5e1',
};

interface Props {
    selectedMatiereId: number | null;
    onSelectMatiere: (m: SelectedMatiere) => void;
}

export function EvalSidebar({ selectedMatiereId, onSelectMatiere }: Props) {
    const theme = useTheme();

    const {
        annees, anneesLoading, annee, setSelectedAnnee,
        tree, treeLoading, selectedPromoId, setSelectedPromoId, selectedPromo,
    } = useAnneePromo();

    const [expandedPeriodeId, setExpandedPeriodeId] = useState<number | null>(null);
    useEffect(() => {
        setExpandedPeriodeId(selectedPromo?.periodes?.[0]?.id ?? null);
    }, [selectedPromo]);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            bgcolor: theme.palette.mode === 'dark' ? '#1e293b' : '#f1f5f9',
        }}>
            {/* En-tête */}
            <Box sx={{ p: 2, borderBottom: '0.5px solid', borderColor: theme.palette.divider, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <AnneePromoSelect
                    annees={annees}
                    anneesLoading={anneesLoading}
                    annee={annee}
                    onAnneeChange={setSelectedAnnee}
                    tree={tree}
                    treeLoading={treeLoading}
                    promoId={selectedPromoId}
                    onPromoChange={setSelectedPromoId}
                    fullWidth
                />
            </Box>

            {/* Arbre */}
            <Box sx={{ flex: 1, overflowY: 'auto' }}>
                {treeLoading && (
                    <Box sx={{ display: 'flex', justifyContent: 'center', pt: 3 }}>
                        <CircularProgress size={22} />
                    </Box>
                )}
                {!treeLoading && (selectedPromo?.periodes ?? []).map(periode => (
                    <Accordion
                        key={periode.id}
                        disableGutters
                        elevation={0}
                        expanded={expandedPeriodeId === periode.id}
                        onChange={(_, open) => setExpandedPeriodeId(open ? periode.id : null)}
                        sx={{
                            bgcolor: 'transparent',
                            '&:before': { display: 'none' },
                            borderBottom: '0.5px solid',
                            borderColor: theme.palette.divider,
                        }}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon sx={{ fontSize: 14 }} />}
                            sx={{ minHeight: 32, px: 1.5, py: 0, '& .MuiAccordionSummary-content': { my: 0 } }}
                        >
                            <Typography variant="caption" fontWeight={600} noWrap sx={{ color: '#64748b' }}>
                                {periode.name}
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails sx={{ p: 0 }}>
                            {periode.matieres.map(matiere => {
                                const isSelected = matiere.id === selectedMatiereId;
                                return (
                                    <Box
                                        key={matiere.id}
                                        onClick={() => {
                                            setExpandedPeriodeId(periode.id);
                                            onSelectMatiere({
                                                id: matiere.id,
                                                name: matiere.name,
                                                periodeName: periode.name,
                                                promotionName: selectedPromo!.name,
                                                annee: annee!,
                                            });
                                        }}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: 1,
                                            pl: 4,
                                            pr: 1.5,
                                            py: 0.75,
                                            cursor: 'pointer',
                                            borderLeft: isSelected ? '3px solid #6366f1' : '3px solid transparent',
                                            bgcolor: isSelected
                                                ? (theme.palette.mode === 'dark' ? 'rgba(99,102,241,0.15)' : '#eef2ff')
                                                : 'transparent',
                                            '&:hover': {
                                                bgcolor: theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.05)' : '#e2e8f0',
                                            },
                                        }}
                                    >
                                        <Box sx={{
                                            width: 8,
                                            height: 8,
                                            borderRadius: '50%',
                                            bgcolor: DOT_COLOR[matiere.dot_status],
                                            flexShrink: 0,
                                        }} />
                                        <Typography
                                            variant="caption"
                                            noWrap
                                            sx={{ color: isSelected ? '#6366f1' : 'text.primary', fontWeight: isSelected ? 600 : 400 }}
                                        >
                                            {matiere.name}
                                        </Typography>
                                    </Box>
                                );
                            })}
                        </AccordionDetails>
                    </Accordion>
                ))}
            </Box>

            {/* Légende */}
            <Box sx={{ p: 1.5, borderTop: '0.5px solid', borderColor: theme.palette.divider, display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
                {[['OK', '#10b981', '≥ 5'], ['WARN', '#f59e0b', '1-4'], ['NONE', '#cbd5e1', '0']].map(([, color, label]) => (
                    <Box key={label} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: color }} />
                        <Typography variant="caption" sx={{ color: '#94a3b8', fontSize: 10 }}>{label}</Typography>
                    </Box>
                ))}
            </Box>
        </Box>
    );
}
