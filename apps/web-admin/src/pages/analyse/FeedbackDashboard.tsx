import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Alert, Box, CircularProgress, MenuItem, Select, Tab, Tabs, Typography } from '@mui/material';
import { apiInstance } from '../../services/api';
import { ENDPOINT_FEEDBACK } from '../feedback/def';
import type { Feedback } from '../feedback/Feedback';
import { UrgencesTab } from './UrgencesTab';
import { TendancesTab } from './TendancesTab';
import { PromotionsTab } from './PromotionsTab';

const MONTH_OPTIONS = [
    { value: 1, label: '1 mois' },
    { value: 2, label: '2 mois' },
    { value: 3, label: '3 mois' },
    { value: 6, label: '6 mois' },
    { value: 12, label: '1 an' },
];

function useFeedbacks(months: number) {
    return useQuery<Feedback[]>({
        queryKey: ['feedback-dashboard', months],
        queryFn: async () => {
            const res = await apiInstance.get<Feedback[]>(`${ENDPOINT_FEEDBACK}/recent?months=${months}`);
            return res.data;
        },
        staleTime: 60_000,
    });
}

export function FeedbackDashboard() {
    const [tab, setTab] = useState(0);
    const [months, setMonths] = useState(() => {
        const saved = localStorage.getItem('dashboard.months');
        return saved ? Number(saved) : 1;
    });
    const handleMonthsChange = (value: number) => {
        setMonths(value);
        localStorage.setItem('dashboard.months', String(value));
    };
    const { data, isLoading, isError } = useFeedbacks(months);

    if (isLoading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" minHeight={300}>
                <CircularProgress />
            </Box>
        );
    }

    if (isError || !data) {
        return (
            <Alert severity="error" sx={{ m: 2 }}>
                Impossible de charger les feedbacks. Vérifiez votre connexion.
            </Alert>
        );
    }

    return (
        <Box sx={{ p: { xs: 1.5, sm: 3 }, maxWidth: 960, mx: 'auto' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <Typography variant="h5" fontWeight={700}>
                    Tableau de bord — Feedbacks étudiants
                </Typography>
                <Select
                    value={months}
                    onChange={e => handleMonthsChange(Number(e.target.value))}
                    variant="standard"
                    disableUnderline
                    sx={{
                        fontSize: '0.82rem',
                        color: 'text.secondary',
                        '& .MuiSelect-select': { py: 0, pr: '20px !important' },
                        '& .MuiSelect-icon': { fontSize: '1rem', top: 'calc(50% - 8px)' },
                    }}
                >
                    {MONTH_OPTIONS.map(o => (
                        <MenuItem key={o.value} value={o.value} sx={{ fontSize: '0.8rem' }}>
                            {o.label}
                        </MenuItem>
                    ))}
                </Select>
            </Box>

            <Tabs
                value={tab}
                onChange={(_, v) => setTab(v)}
                sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
            >
                <Tab label="Urgences" />
                <Tab label="Tendances" />
                <Tab label="Promotions" />
            </Tabs>

            {tab === 0 && <UrgencesTab data={data} />}
            {tab === 1 && <TendancesTab data={data} />}
            {tab === 2 && <PromotionsTab data={data} />}
        </Box>
    );
}
