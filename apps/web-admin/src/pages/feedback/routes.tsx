import { createCrudRoutes, type CrudComponentProps } from '../../services/crud/routes';
import { FEEDBACK, FEEDBACK_WORKFLOW } from './def';
import { CrudFeedback } from './Feedback';
import { Box } from '@mui/material';
import type { ReactNode } from 'react';

export function createFeedbackRoutes() {
    return [
        createCrudRoutes(FEEDBACK, CustomCrudFeedback),
    ];
}

function CustomCrudFeedback({ mode }: CrudComponentProps) {
    const renderTopToolbar = ({ defaultActions }: { defaultActions: ReactNode; isEditMode: boolean }) => (
        <Box sx={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            {defaultActions}
        </Box>
    );

    return (
        <CrudFeedback
            workflow={FEEDBACK_WORKFLOW}
            mode={mode}
            isAction={false}
            isTopToolbar={true}
            renderTopToolbarCustomActions={renderTopToolbar}
        />
    );
}
