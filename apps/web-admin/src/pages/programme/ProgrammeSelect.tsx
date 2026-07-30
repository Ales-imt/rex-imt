import { useNavigate, Navigate } from 'react-router';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Button from '@mui/material/Button';
import { useAnneePromo } from '../../hooks/useCurriculum';
import { AnneePromoSelect } from '../../components/AnneePromoSelect';
import { PROGRAMME_WORKFLOW, PROGRAMME_LAST_PERIODE_KEY } from './def';

// Route index /programme : reprend le dernier planning consulté s'il existe,
// sinon bascule sur la page de sélection.
export function ProgrammeIndex() {
    const last = sessionStorage.getItem(PROGRAMME_LAST_PERIODE_KEY);
    return (
        <Navigate
            to={last ? `/${PROGRAMME_WORKFLOW}/${last}` : `/${PROGRAMME_WORKFLOW}/select`}
            replace
        />
    );
}

export function ProgrammeSelect() {
    const navigate = useNavigate();

    const {
        annees, anneesLoading, annee, setSelectedAnnee,
        tree, treeLoading, selectedPromoId, setSelectedPromoId, selectedPromo,
        selectedPeriodeId, setSelectedPeriodeId,
    } = useAnneePromo('programme');

    return (
        <Box sx={{ p: 3, maxWidth: 720, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={3}>Programme des cours</Typography>

            <Paper elevation={0} sx={{ p: 2, border: '1px solid', borderColor: 'divider', display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
                <AnneePromoSelect
                    annees={annees}
                    anneesLoading={anneesLoading}
                    annee={annee}
                    onAnneeChange={setSelectedAnnee}
                    tree={tree}
                    treeLoading={treeLoading}
                    promoId={selectedPromoId}
                    onPromoChange={setSelectedPromoId}
                />

                {selectedPromo && (
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>Période</InputLabel>
                        <Select
                            value={selectedPeriodeId ?? ''}
                            label="Période"
                            onChange={e => setSelectedPeriodeId(Number(e.target.value))}
                        >
                            {selectedPromo.periodes.map(p => (
                                <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                )}

                <Button
                    variant="contained"
                    disabled={selectedPeriodeId == null}
                    onClick={() => navigate(`/${PROGRAMME_WORKFLOW}/${selectedPeriodeId}`)}
                >
                    Afficher le planning
                </Button>
            </Paper>
        </Box>
    );
}
