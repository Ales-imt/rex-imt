import { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import CircularProgress from '@mui/material/CircularProgress';
import { useTheme } from '@mui/material/styles';
import InboxIcon from '@mui/icons-material/Inbox';
import { useMatiereStats } from './useEvaluations';
import { MetricCards } from './components/MetricCards';
import { ScoreBars } from './components/ScoreBars';
import { NpsBar } from './components/NpsBar';
import { ChipSignals } from './components/ChipSignals';
import { VerbatimList } from './components/VerbatimList';
import { SyntheseIATab } from './components/SyntheseIATab';
import type { SelectedMatiere, SyntheseIA } from './Evaluation';

const STATUT_COLOR: Record<string, { bg: string; color: string }> = {
    GENEREE:  { bg: '#eff6ff', color: '#1d4ed8' },
    VALIDEE:  { bg: '#f0fdf4', color: '#15803d' },
    ARCHIVEE: { bg: '#f8fafc', color: '#64748b' },
};

interface Props {
    selected: SelectedMatiere | null;
}

function ContextPills({ stats }: { stats: { label: string; pct: number }[] }) {
    const theme = useTheme();
    return (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {stats.map(({ label, pct }) => (
                <Box key={label} sx={{
                    bgcolor: theme.palette.action.hover,
                    border: '0.5px solid',
                    borderColor: theme.palette.divider,
                    borderRadius: '6px',
                    px: 1.5,
                    py: 0.5,
                    display: 'flex',
                    gap: 0.75,
                    alignItems: 'center',
                }}>
                    <Typography variant="caption" fontWeight={500}>{label}</Typography>
                    <Typography variant="caption" sx={{ color: '#64748b' }}>{pct.toFixed(0)}%</Typography>
                </Box>
            ))}
        </Box>
    );
}

export function EvalDetail({ selected }: Props) {
    const theme = useTheme();
    const [tab, setTab] = useState(0);
    const [localSynthese, setLocalSynthese] = useState<SyntheseIA | null>(null);

    const { data: stats, loading } = useMatiereStats(selected?.id ?? null);

    useEffect(() => { setLocalSynthese(null); }, [selected?.id]);

    if (!selected) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', gap: 2, color: '#94a3b8' }}>
                <InboxIcon sx={{ fontSize: 48, opacity: 0.4 }} />
                <Typography variant="body2">Sélectionnez une matière dans le menu</Typography>
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* Section 1 — En-tête */}
            <Box>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                    {selected.annee}/{selected.annee + 1} › {selected.promotionName} › {selected.periodeName}
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mt: 0.5, flexWrap: 'wrap' }}>
                    <Typography variant="h5" fontWeight={700} sx={{ color: 'text.primary' }}>
                        {selected.name}
                    </Typography>
                    {(localSynthese ?? stats?.synthese) ? (
                        <Box sx={{
                            bgcolor: STATUT_COLOR[(localSynthese ?? stats!.synthese!).statut]?.bg ?? '#f8fafc',
                            color: STATUT_COLOR[(localSynthese ?? stats!.synthese!).statut]?.color ?? '#64748b',
                            borderRadius: '6px',
                            px: 1.5,
                            py: 0.25,
                        }}>
                            <Typography variant="caption" fontWeight={600}>{(localSynthese ?? stats!.synthese!).statut}</Typography>
                        </Box>
                    ) : (
                        <Box sx={{ bgcolor: theme.palette.action.hover, borderRadius: '6px', px: 1.5, py: 0.25 }}>
                            <Typography variant="caption" sx={{ color: '#94a3b8' }}>Aucune synthèse</Typography>
                        </Box>
                    )}
                </Box>
            </Box>

            {loading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
                    <CircularProgress />
                </Box>
            )}

            {!loading && stats && (
                <>
                    {/* Section 2 — Métriques */}
                    <MetricCards stats={stats} />

                    {/* Section 3 — Onglets */}
                    <Box sx={{ borderBottom: '0.5px solid', borderColor: theme.palette.divider }}>
                        <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 40 }}>
                            <Tab label="Vue d'ensemble" sx={{ minHeight: 40, py: 0 }} />
                            <Tab label="Verbatims" sx={{ minHeight: 40, py: 0 }} />
                            <Tab label="Synthèse IA" sx={{ minHeight: 40, py: 0 }} />
                        </Tabs>
                    </Box>

                    {tab === 0 && (
                        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                            <ScoreBars stats={stats} />

                            {stats.nps_distribution && <NpsBar distribution={stats.nps_distribution} />}

                            {stats.chip_stats.length > 0 && <ChipSignals chips={stats.chip_stats} />}

                            {/* Distribution contexte */}
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
                                <Typography variant="subtitle2" fontWeight={700}>Distribution contexte</Typography>
                                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                                    <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 500 }}>Format suivi</Typography>
                                    <ContextPills stats={[
                                        { label: 'Présentiel',  pct: stats.pct_format_presentiel },
                                        { label: 'Distanciel', pct: stats.pct_format_distanciel },
                                        { label: 'Hybride',    pct: stats.pct_format_hybride },
                                    ].filter(s => s.pct > 0)} />
                                </Box>
                                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                                    <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 500 }}>Tendance semestre</Typography>
                                    <ContextPills stats={[
                                        { label: 'En progrès', pct: stats.pct_tendance_progres },
                                        { label: 'Stable',     pct: stats.pct_tendance_stable },
                                        { label: 'En baisse',  pct: stats.pct_tendance_baisse },
                                    ].filter(s => s.pct > 0)} />
                                </Box>
                            </Box>
                        </Box>
                    )}

                    {tab === 1 && <VerbatimList key={selected.id} matiereId={selected.id} />}

                    {tab === 2 && (
                        <SyntheseIATab
                            synthese={localSynthese ?? stats.synthese}
                            nbRepondants={stats.nb_repondants}
                            matiereId={selected.id}
                            onSyntheseChange={setLocalSynthese}
                        />
                    )}
                </>
            )}
        </Box>
    );
}
