import { useEffect } from 'react';
import { Outlet, useLocation, Navigate } from 'react-router';
import { Box } from '@mui/material';
import { FEEDBACK, FEEDBACK_WORKFLOW } from './def';

const STORAGE_KEY = 'feedback_last_path';

export function FeedbackLayout() {
    const location = useLocation();

    useEffect(() => {
        if (!location.pathname.endsWith(`/${FEEDBACK_WORKFLOW}`)) {
            sessionStorage.setItem(STORAGE_KEY, location.pathname);
        }
    }, [location]);

    return (
        <Box sx={{ flex: 1, overflow: 'auto' }}>
            <Outlet />
        </Box>
    );
}

export function FeedbackIndex() {
    const lastPath = sessionStorage.getItem(STORAGE_KEY);
    const target = lastPath || FEEDBACK;

    return <Navigate to={target} replace />;
}
