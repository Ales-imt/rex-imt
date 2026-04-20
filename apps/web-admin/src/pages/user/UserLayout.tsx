import { useEffect } from 'react';
import { Outlet, useLocation, Navigate } from 'react-router';
import { Box } from '@mui/material';
import { USER, USER_WORKFLOW } from './def';

const STORAGE_KEY = 'user_last_path';



export function UserLayout() {
    const location = useLocation();

    useEffect(() => {
        if (!location.pathname.endsWith(`/${USER_WORKFLOW}`)) {
            sessionStorage.setItem(STORAGE_KEY, location.pathname);
        }
    }, [location]);

    return (
        <Box sx={{ flex: 1, overflow: 'auto' }}>
            <Outlet />
        </Box>
    );
}

export function UserIndex() {
    const lastPath = sessionStorage.getItem(STORAGE_KEY);
    const target = lastPath || USER;

    return <Navigate to={target} replace />;
}
