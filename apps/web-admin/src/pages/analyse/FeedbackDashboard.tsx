import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Alert, Box, CircularProgress, Tab, Tabs, Typography } from '@mui/material';
import { apiInstance } from '../../services/api';
import { ENDPOINT_FEEDBACK } from '../feedback/def';
import type { Feedback } from '../feedback/Feedback';
import { UrgencesTab } from './UrgencesTab';
import { TendancesTab } from './TendancesTab';
import { PromotionsTab } from './PromotionsTab';

function useFeedbacks() {
    return useQuery<Feedback[]>({
        queryKey: ['feedback-dashboard'],
        queryFn: async () => {
            const res = await apiInstance.get<Feedback[]>(ENDPOINT_FEEDBACK);
            return res.data;
        },
        staleTime: 60_000,
    });
}

export function FeedbackDashboard() {
    const [tab, setTab] = useState(0);
    const { data, isLoading, isError } = useFeedbacks();

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
            <Typography variant="h5" fontWeight={700} mb={2}>
                Tableau de bord — Feedbacks étudiants
            </Typography>

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
