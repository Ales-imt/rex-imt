import { useMemo, useState } from 'react';
import {
    Box,
    Button,
    Chip,
    CircularProgress,
    Divider,
    LinearProgress,
    Paper,
    Typography,
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import dayjs, { type Dayjs } from 'dayjs';
import type { Feedback } from '../feedback/Feedback';
import { apiInstance } from '../../services/api';
import { KpiCard } from './shared';

type ReportState = 'idle' | 'loading' | 'ok' | 'error';

async function downloadReport(endpoint: string, body: object | null, filename: string) {
    const res = await apiInstance.post(endpoint, body, { responseType: 'blob' });
    const url = URL.createObjectURL(new Blob([res.data], { type: 'application/pdf' }));
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
}

export function PromotionsTab({ data }: { data: Feedback[] }) {
    const [weeklyState, setWeeklyState] = useState<ReportState>('idle');
    const [periodState, setPeriodState] = useState<ReportState>('idle');
    const [from, setFrom] = useState<Dayjs | null>(dayjs().subtract(30, 'day'));
    const [to, setTo] = useState<Dayjs | null>(dayjs());

    const byPromotion = useMemo(() => {
        const map = new Map<string, { total: number; hasCritical: boolean }>();
        data.forEach(f => {
            const key = f.promotion ?? 'Promotion inconnue';
            const prev = map.get(key) ?? { total: 0, hasCritical: false };
            map.set(key, { total: prev.total + 1, hasCritical: prev.hasCritical || f.urgence === 5 });
        });
        return Array.from(map.entries()).sort((a, b) => b[1].total - a[1].total);
    }, [data]);

    const maxTotal = byPromotion[0]?.[1].total ?? 1;
    const mostActive = byPromotion[0]?.[0] ?? '—';
    const safeCount = byPromotion.filter(([, v]) => !v.hasCritical).length;

    const handleWeekly = async () => {
        setWeeklyState('loading');
        try {
            await downloadReport('/api/v2/reports/weekly', null, `rapport-hebdo-${dayjs().format('YYYY-MM-DD')}.pdf`);
            setWeeklyState('ok');
        } catch {
            setWeeklyState('error');
        }
    };

    const handlePeriod = async () => {
        if (!from || !to) return;
        setPeriodState('loading');
        try {
            await downloadReport(
                '/api/v2/reports/period',
                { from: from.format('YYYY-MM-DD'), to: to.format('YYYY-MM-DD') },
                `rapport-${from.format('YYYY-MM-DD')}_${to.format('YYYY-MM-DD')}.pdf`,
            );
            setPeriodState('ok');
        } catch {
            setPeriodState('error');
        }
    };

    return (
        <Box>
            <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
                <KpiCard label="Promotion la plus active" value={mostActive} color="#1976d2" />
                <KpiCard label="Promotions sans alerte critique" value={safeCount} color="#2e7d32" />
            </Box>

            <Paper elevation={1} sx={{ p: 2.5, mb: 3 }}>
                <Typography variant="subtitle1" fontWeight={600} mb={2}>Volume par promotion</Typography>
                {byPromotion.length === 0
                    ? <Typography variant="body2" color="text.secondary">Aucune donnée</Typography>
                    : byPromotion.map(([promo, { total, hasCritical }]) => (
                        <Box key={promo} sx={{ mb: 1.5 }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5, alignItems: 'center' }}>
                                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                    <Typography variant="body2">{promo}</Typography>
                                    {hasCritical
                                        ? <Chip label="Alerte" size="small" color="error" sx={{ height: 18, fontSize: '0.65rem' }} />
                                        : <Chip label="OK" size="small" color="success" sx={{ height: 18, fontSize: '0.65rem' }} />
                                    }
                                </Box>
                                <Typography variant="body2" color="text.secondary">{total}</Typography>
                            </Box>
                            <LinearProgress
                                variant="determinate"
                                value={Math.round((total / maxTotal) * 100)}
                                sx={{
                                    height: 8,
                                    borderRadius: 1,
                                    bgcolor: '#f0f0f0',
                                    '& .MuiLinearProgress-bar': {
                                        bgcolor: hasCritical ? '#d32f2f' : '#1976d2',
                                        borderRadius: 1,
                                    },
                                }}
                            />
                        </Box>
                    ))
                }
            </Paper>

            <Divider sx={{ mb: 2 }} />

            {/* Rapport hebdomadaire */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap', mb: 3 }}>
                <Button
                    variant="contained"
                    onClick={handleWeekly}
                    disabled={weeklyState === 'loading'}
                    startIcon={weeklyState === 'loading' ? <CircularProgress size={16} color="inherit" /> : undefined}
                >
                    Rapport des 7 derniers jours
                </Button>
                {weeklyState === 'ok' && <Typography variant="body2" color="success.main">Téléchargement lancé.</Typography>}
                {weeklyState === 'error' && <Typography variant="body2" color="error">Erreur lors de la génération.</Typography>}
            </Box>

            {/* Rapport sur période */}
            <Paper elevation={1} sx={{ p: 2.5 }}>
                <Typography variant="subtitle1" fontWeight={600} mb={2}>Rapport sur une période personnalisée</Typography>
                <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
                    <DatePicker
                        label="Du"
                        value={from}
                        onChange={setFrom}
                        maxDate={to ?? undefined}
                        slotProps={{ textField: { size: 'small' } }}
                    />
                    <DatePicker
                        label="Au"
                        value={to}
                        onChange={setTo}
                        minDate={from ?? undefined}
                        maxDate={dayjs()}
                        slotProps={{ textField: { size: 'small' } }}
                    />
                    <Button
                        variant="outlined"
                        onClick={handlePeriod}
                        disabled={!from || !to || periodState === 'loading'}
                        startIcon={periodState === 'loading' ? <CircularProgress size={16} color="inherit" /> : undefined}
                    >
                        Générer le rapport
                    </Button>
                </Box>
                {periodState === 'ok' && <Typography variant="body2" color="success.main" mt={1}>Téléchargement lancé.</Typography>}
                {periodState === 'error' && <Typography variant="body2" color="error" mt={1}>Erreur lors de la génération.</Typography>}
            </Paper>
        </Box>
    );
}
