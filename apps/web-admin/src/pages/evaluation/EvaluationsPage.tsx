import { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import { AnneePromoSelect } from '../../components/AnneePromoSelect';
import { useAnneePromo } from '../../hooks/useCurriculum';
import { EvalDetail } from './EvalDetail';
import type { SelectedMatiere } from './Evaluation';

export function EvaluationsPage() {
    const {
        annees, anneesLoading, annee, setSelectedAnnee,
        tree, treeLoading, selectedPromoId, setSelectedPromoId, selectedPromo,
        selectedPeriodeId, setSelectedPeriodeId, selectedPeriode,
        selectedMatiereId, setSelectedMatiereId,
    } = useAnneePromo('evaluation');

    const [selectedMatiere, setSelectedMatiere] = useState<SelectedMatiere | null>(null);
    useEffect(() => {
        if (!selectedMatiereId || !selectedPeriode || !selectedPromo || !annee) {
            setSelectedMatiere(null);
            return;
        }
        const matiere = selectedPeriode.matieres.find(m => m.id === selectedMatiereId);
        if (!matiere) return;
        setSelectedMatiere({
            id: matiere.id,
            name: matiere.name,
            periodeName: selectedPeriode.name,
            promotionName: selectedPromo.name,
            annee,
        });
    }, [selectedMatiereId, selectedPeriode, selectedPromo, annee]);

    return (
        <Box sx={{ p: 3, maxWidth: 960, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={3}>Évaluations</Typography>

            <Paper elevation={0} sx={{ p: 2, mb: 3, border: '1px solid', borderColor: 'divider', display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
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
                    <FormControl size="small" sx={{ minWidth: 160 }}>
                        <InputLabel>Semestre</InputLabel>
                        <Select
                            value={selectedPeriodeId ?? ''}
                            label="Semestre"
                            onChange={e => setSelectedPeriodeId(Number(e.target.value))}
                        >
                            {selectedPromo.periodes.map(p => (
                                <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                )}

                {selectedPeriode && (
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>Matière</InputLabel>
                        <Select
                            value={selectedMatiereId ?? ''}
                            label="Matière"
                            onChange={e => setSelectedMatiereId(Number(e.target.value))}
                            sx={{ '& .MuiSelect-select': { color: selectedMatiereId ? '#4f46e5' : undefined, fontWeight: selectedMatiereId ? 600 : undefined } }}
                        >
                            {selectedPeriode.matieres.map(m => (
                                <MenuItem key={m.id} value={m.id}>{m.name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                )}
            </Paper>

            <EvalDetail selected={selectedMatiere} />
        </Box>
    );
}
