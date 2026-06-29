import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import CircularProgress from '@mui/material/CircularProgress';
import type { PromotionTree } from '../hooks/useCurriculum';

interface Props {
    annees: number[] | null;
    anneesLoading: boolean;
    annee: number | null;
    onAnneeChange: (a: number) => void;
    tree: PromotionTree[] | null;
    treeLoading: boolean;
    promoId: number | null;
    onPromoChange: (id: number) => void;
    fullWidth?: boolean;
}

export function AnneePromoSelect({
    annees,
    anneesLoading,
    annee,
    onAnneeChange,
    tree,
    treeLoading,
    promoId,
    onPromoChange,
    fullWidth = false,
}: Props) {
    const minWidth = fullWidth ? undefined : 110;

    return (
        <>
            {anneesLoading ? (
                <CircularProgress size={18} />
            ) : (
                <FormControl size="small" fullWidth={fullWidth} sx={fullWidth ? undefined : { minWidth }}>
                    <InputLabel>Année</InputLabel>
                    <Select
                        value={annee ?? ''}
                        label="Année"
                        onChange={e => onAnneeChange(Number(e.target.value))}
                    >
                        {(annees ?? []).map(a => (
                            <MenuItem key={a} value={a}>{a}/{a + 1}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
            )}

            {!treeLoading && (tree ?? []).length > 0 && (
                <FormControl size="small" fullWidth={fullWidth} sx={fullWidth ? undefined : { minWidth: 160 }}>
                    <InputLabel>Promotion</InputLabel>
                    <Select
                        value={promoId ?? ''}
                        label="Promotion"
                        onChange={e => onPromoChange(Number(e.target.value))}
                    >
                        {(tree ?? []).map(p => (
                            <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
            )}
        </>
    );
}
