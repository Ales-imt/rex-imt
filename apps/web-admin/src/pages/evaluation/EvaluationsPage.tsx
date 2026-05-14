import { useState } from 'react';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import { EvalSidebar } from './EvalSidebar';
import { EvalDetail } from './EvalDetail';
import type { SelectedMatiere } from './Evaluation';

export function EvaluationsPage() {
    const theme = useTheme();
    const [selectedMatiere, setSelectedMatiere] = useState<SelectedMatiere | null>(null);

    return (
        <Box sx={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
            <Box sx={{
                width: 240,
                flexShrink: 0,
                borderRight: '0.5px solid',
                borderColor: theme.palette.divider,
                overflow: 'hidden',
                display: 'flex',
                flexDirection: 'column',
            }}>
                <EvalSidebar
                    selectedMatiereId={selectedMatiere?.id ?? null}
                    onSelectMatiere={setSelectedMatiere}
                />
            </Box>
            <Box sx={{ flex: 1, overflowY: 'auto', bgcolor: theme.palette.mode === 'dark' ? 'background.default' : '#f8fafc' }}>
                <EvalDetail selected={selectedMatiere} />
            </Box>
        </Box>
    );
}
