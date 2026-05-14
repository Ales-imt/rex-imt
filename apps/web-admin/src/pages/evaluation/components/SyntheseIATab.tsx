import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Tooltip from '@mui/material/Tooltip';
import CircularProgress from '@mui/material/CircularProgress';
import Alert from '@mui/material/Alert';
import { useTheme } from '@mui/material/styles';
import { useGenerateSynthese, useValidateSynthese } from '../useEvaluations';
import { useSession } from '../../../SessionContext';
import type { SyntheseIA } from '../Evaluation';

const STATUT_COLOR: Record<string, { bg: string; color: string }> = {
    GENEREE:  { bg: '#eff6ff', color: '#1d4ed8' },
    VALIDEE:  { bg: '#f0fdf4', color: '#15803d' },
    ARCHIVEE: { bg: '#f8fafc', color: '#64748b' },
};

interface Props {
    synthese: SyntheseIA | null;
    nbRepondants: number;
    matiereId: number;
    onSyntheseChange: (s: SyntheseIA) => void;
}

export function SyntheseIATab({ synthese, nbRepondants, matiereId, onSyntheseChange }: Props) {
    const theme = useTheme();
    const { session } = useSession();

    const { generate, loading: generating, error: genError } = useGenerateSynthese(matiereId);
    const { validate, loading: validating } = useValidateSynthese();

    const current = synthese;
    const canGenerate = nbRepondants >= 5;

    // État 1 — pas assez de répondants
    if (!canGenerate) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, py: 6 }}>
                <Typography variant="body2" sx={{ color: '#64748b' }}>
                    Il faut au moins 5 répondants pour générer une synthèse IA.
                </Typography>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                    Actuellement : {nbRepondants} répondant{nbRepondants > 1 ? 's' : ''}
                </Typography>
            </Box>
        );
    }

    // État 3 — génération en cours
    if (generating) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, py: 6 }}>
                <CircularProgress size={32} sx={{ color: '#6366f1' }} />
                <Typography variant="body2" sx={{ color: '#64748b' }}>
                    Génération en cours…
                </Typography>
            </Box>
        );
    }

    // État 4 — erreur de génération
    if (genError) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, py: 2 }}>
                <Alert severity="error">{genError}</Alert>
                <Box>
                    <Button
                        variant="contained"
                        onClick={async () => {
                            const result = await generate();
                            if (result) onSyntheseChange(result);
                        }}
                        sx={{ bgcolor: '#6366f1', '&:hover': { bgcolor: '#4f46e5' } }}
                    >
                        Réessayer
                    </Button>
                </Box>
            </Box>
        );
    }

    // État 2 — aucune synthèse
    if (!current) {
        return (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, py: 6 }}>
                <Typography variant="body2" sx={{ color: '#64748b' }}>
                    Aucune synthèse IA disponible pour cette matière.
                </Typography>
                <Button
                    variant="contained"
                    onClick={async () => {
                        const result = await generate();
                        if (result) onSyntheseChange(result);
                    }}
                    sx={{ bgcolor: '#6366f1', '&:hover': { bgcolor: '#4f46e5' } }}
                >
                    Générer la synthèse IA
                </Button>
            </Box>
        );
    }

    // État 5 — synthèse disponible
    const statut = STATUT_COLOR[current.statut] ?? STATUT_COLOR.GENEREE;
    const date = new Date(current.generated_at).toLocaleDateString('fr-FR', {
        day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
    });

    const handleValidate = async () => {
        const email = session?.user?.email ?? '';
        const ok = await validate(current.id, email);
        if (ok) onSyntheseChange({ ...current, statut: 'VALIDEE', validated_by: email });
    };

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
            <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start', flexWrap: 'wrap' }}>
                <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" sx={{ color: '#64748b' }}>
                        Générée le {date} · {current.nb_repondants} répondants
                    </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                    <Box sx={{ bgcolor: statut.bg, color: statut.color, borderRadius: '6px', px: 1.5, py: 0.5 }}>
                        <Typography variant="caption" fontWeight={600}>{current.statut}</Typography>
                    </Box>
                    <Tooltip title="Générer une nouvelle synthèse" arrow>
                        <span>
                            <Button
                                size="small"
                                variant="outlined"
                                disabled={generating}
                                onClick={async () => {
                                    const result = await generate();
                                    if (result) onSyntheseChange(result);
                                }}
                            >
                                Régénérer
                            </Button>
                        </span>
                    </Tooltip>
                    {current.statut === 'GENEREE' && (
                        <Button
                            size="small"
                            variant="contained"
                            disabled={validating}
                            onClick={handleValidate}
                            sx={{ bgcolor: '#15803d', '&:hover': { bgcolor: '#166534' } }}
                        >
                            {validating ? <CircularProgress size={16} sx={{ color: '#fff' }} /> : 'Valider'}
                        </Button>
                    )}
                </Box>
            </Box>

            <Box sx={{
                bgcolor: theme.palette.background.paper,
                border: '0.5px solid',
                borderColor: theme.palette.divider,
                borderRadius: '12px',
                p: 2.5,
                maxHeight: 400,
                overflowY: 'auto',
            }}>
                <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.8, color: 'text.primary' }}>
                    {current.synthese_texte}
                </Typography>
            </Box>
        </Box>
    );
}
