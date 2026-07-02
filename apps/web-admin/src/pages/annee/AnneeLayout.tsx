import { useEffect } from 'react';
import { Outlet, useLocation, Navigate } from 'react-router';
import { Box } from '@mui/material';
import { ANNEE, ANNEE_WORKFLOW } from './def';

const STORAGE_KEY = 'annee_last_path';



export function AnneeLayout() {
    const location = useLocation();

    useEffect(() => {
        if (!location.pathname.endsWith(`/${ANNEE_WORKFLOW}`)) {
            sessionStorage.setItem(STORAGE_KEY, location.pathname);
        }
    }, [location]);

    return (
        <Box sx={{ flex: 1, overflow: 'auto' }}>
            <Outlet />
        </Box>
    );
}

export function AnneeIndex() {
    const lastPath = sessionStorage.getItem(STORAGE_KEY);
    const target = lastPath || ANNEE;

    return <Navigate to={target} replace />;
}
