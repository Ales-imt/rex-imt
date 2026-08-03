import { useState, type ReactNode } from 'react';
import Box from '@mui/material/Box';
import Alert from '@mui/material/Alert';
import Badge from '@mui/material/Badge';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import CardActions from '@mui/material/CardActions';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import MenuItem from '@mui/material/MenuItem';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Stack from '@mui/material/Stack';
import { useQuery, useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { apiInstance } from '../../services/api';
import {
    DIMENSION_LABEL,
    ENDPOINT_MODERATION,
    ENDPOINT_MODERATION_VERBATIM,
    PROMOTION_ALL,
    PROMOTION_NONE,
    type PendingFeedback,
    type PendingVerbatim,
} from './def';

const QUERY_KEY_FEEDBACK: QueryKey = ['moderation-pending', 'feedback'];
const QUERY_KEY_VERBATIM: QueryKey = ['moderation-pending', 'verbatim'];

function usePendingList<T>(queryKey: QueryKey, url: string) {
    return useQuery<T[]>({
        queryKey,
        queryFn: async () => {
            const res = await apiInstance.get<T[]>(url);
            return res.data;
        },
        staleTime: 15_000,
        refetchInterval: 30_000,
    });
}

type Section = 'feedback' | 'verbatim';

type Pending = { promotion?: string };

function matchesPromotion(item: Pending, promotion: string): boolean {
    if (promotion === PROMOTION_ALL) return true;
    if (promotion === PROMOTION_NONE) return !item.promotion;
    return item.promotion === promotion;
}

export function Moderation() {
    const [section, setSection] = useState<Section>('feedback');
    const [promotion, setPromotion] = useState<string>(PROMOTION_ALL);

    const feedbacks = usePendingList<PendingFeedback>(
        QUERY_KEY_FEEDBACK,
        `${ENDPOINT_MODERATION}/pending`,
    );
    const verbatims = usePendingList<PendingVerbatim>(
        QUERY_KEY_VERBATIM,
        `${ENDPOINT_MODERATION_VERBATIM}/pending`,
    );

    const current = section === 'feedback' ? feedbacks : verbatims;

    // Options bâties sur l'UNION des deux files : la promotion sélectionnée
    // reste ainsi valide quand on change d'onglet, sans réinitialisation.
    const allItems: Pending[] = [...(feedbacks.data ?? []), ...(verbatims.data ?? [])];
    const promotionNames = [...new Set(allItems.map((i) => i.promotion).filter(Boolean) as string[])]
        .sort((a, b) => a.localeCompare(b, 'fr'));
    const hasUnassigned = allItems.some((i) => !i.promotion);

    // Une promotion entièrement modérée disparaît des options : on retombe sur
    // « Toutes » plutôt que de laisser le Select sur une valeur inexistante.
    const selectable =
        promotion === PROMOTION_ALL ||
        (promotion === PROMOTION_NONE && hasUnassigned) ||
        promotionNames.includes(promotion);
    const activePromotion = selectable ? promotion : PROMOTION_ALL;

    const visibleFeedbacks = (feedbacks.data ?? []).filter((i) => matchesPromotion(i, activePromotion));
    const visibleVerbatims = (verbatims.data ?? []).filter((i) => matchesPromotion(i, activePromotion));

    return (
        <Box sx={{ p: 3, maxWidth: 900, mx: 'auto' }}>
            <Typography variant="h5" sx={{ mb: 1 }}>
                Modération des messages
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Ces messages ne sont ni diffusés ni analysés tant qu'ils n'ont pas été publiés.
                Vous pouvez corriger le texte avant publication. L'auteur n'est pas identifiable.
            </Typography>

            <Tabs
                value={section}
                onChange={(_, v: Section) => setSection(v)}
                sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}
            >
                <Tab
                    value="feedback"
                    label={<TabLabel text="Feedbacks" count={visibleFeedbacks.length} />}
                />
                <Tab
                    value="verbatim"
                    label={<TabLabel text="Verbatims d'évaluation" count={visibleVerbatims.length} />}
                />
            </Tabs>

            {(promotionNames.length > 0 || hasUnassigned) && (
                <TextField
                    select
                    size="small"
                    label="Promotion"
                    value={activePromotion}
                    onChange={(e) => setPromotion(e.target.value)}
                    sx={{ mb: 3, minWidth: 260 }}
                >
                    <MenuItem value={PROMOTION_ALL}>Toutes les promotions</MenuItem>
                    {promotionNames.map((name) => (
                        <MenuItem key={name} value={name}>
                            {name}
                        </MenuItem>
                    ))}
                    {hasUnassigned && (
                        <MenuItem value={PROMOTION_NONE}>Promotion non renseignée</MenuItem>
                    )}
                </TextField>
            )}

            {current.isError ? (
                <Alert severity="error">
                    {section === 'feedback'
                        ? 'Impossible de charger les messages à modérer.'
                        : 'Impossible de charger les verbatims à modérer.'}
                </Alert>
            ) : current.isLoading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
                    <CircularProgress />
                </Box>
            ) : section === 'feedback' ? (
                <FeedbackList items={visibleFeedbacks} filtered={activePromotion !== PROMOTION_ALL} />
            ) : (
                <VerbatimList items={visibleVerbatims} filtered={activePromotion !== PROMOTION_ALL} />
            )}
        </Box>
    );
}

function TabLabel({ text, count }: { text: string; count?: number }) {
    return (
        <Badge badgeContent={count} color="primary" sx={{ pr: count ? 2 : 0 }}>
            {text}
        </Badge>
    );
}

function FeedbackList({ items, filtered }: { items: PendingFeedback[]; filtered: boolean }) {
    if (items.length === 0) {
        return (
            <Alert severity={filtered ? 'info' : 'success'}>
                {filtered
                    ? 'Aucun message en attente pour cette promotion.'
                    : 'Aucun message en attente de modération.'}
            </Alert>
        );
    }
    return (
        <Stack spacing={2}>
            {items.map((item) => (
                <ModerationCard
                    key={item.id}
                    reference={`#${item.id}`}
                    rawText={item.raw_content}
                    createdAt={item.created_at}
                    promotion={item.promotion}
                    queryKey={QUERY_KEY_FEEDBACK}
                    approve={(texte) =>
                        apiInstance.post(`${ENDPOINT_MODERATION}/${item.id}/approve`, {
                            content_modere: texte,
                        })
                    }
                    reject={(reason) =>
                        apiInstance.post(`${ENDPOINT_MODERATION}/${item.id}/reject`, { reason })
                    }
                />
            ))}
        </Stack>
    );
}

function VerbatimList({ items, filtered }: { items: PendingVerbatim[]; filtered: boolean }) {
    if (items.length === 0) {
        return (
            <Alert severity={filtered ? 'info' : 'success'}>
                {filtered
                    ? 'Aucun verbatim en attente pour cette promotion.'
                    : 'Aucun verbatim en attente de modération.'}
            </Alert>
        );
    }
    return (
        <Stack spacing={2}>
            {items.map((item) => (
                <ModerationCard
                    key={item.id}
                    reference={`Verbatim ${item.id.slice(0, 8)}`}
                    rawText={item.raw_texte}
                    createdAt={item.created_at}
                    promotion={item.promotion}
                    queryKey={QUERY_KEY_VERBATIM}
                    badge={
                        <Chip
                            size="small"
                            label={DIMENSION_LABEL[item.dimension] ?? item.dimension}
                            sx={{ ml: 1 }}
                        />
                    }
                    approve={(texte) =>
                        apiInstance.post(`${ENDPOINT_MODERATION_VERBATIM}/${item.id}/approve`, {
                            texte_modere: texte,
                        })
                    }
                    reject={(reason) =>
                        apiInstance.post(`${ENDPOINT_MODERATION_VERBATIM}/${item.id}/reject`, {
                            reason,
                        })
                    }
                />
            ))}
        </Stack>
    );
}

/** Carte de relecture commune aux feedbacks et aux verbatims : le pipeline de
 *  modération est le même (publier le texte, éventuellement corrigé, ou refuser
 *  avec un motif), seuls l'endpoint et la clé du corps changent. */
function ModerationCard({
    reference,
    rawText,
    createdAt,
    promotion,
    queryKey,
    approve,
    reject,
    badge,
}: {
    reference: string;
    rawText: string;
    createdAt: string;
    promotion?: string;
    queryKey: QueryKey;
    approve: (texte: string) => Promise<unknown>;
    reject: (reason: string) => Promise<unknown>;
    badge?: ReactNode;
}) {
    const queryClient = useQueryClient();
    const [content, setContent] = useState(rawText);
    const [reason, setReason] = useState('');
    const [rejecting, setRejecting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const invalidate = () => queryClient.invalidateQueries({ queryKey });

    const approveMutation = useMutation({
        mutationFn: () => approve(content),
        onSuccess: invalidate,
        onError: () => setError('Échec de la publication.'),
    });

    const rejectMutation = useMutation({
        mutationFn: () => reject(reason),
        onSuccess: invalidate,
        onError: () => setError('Échec du refus.'),
    });

    const busy = approveMutation.isPending || rejectMutation.isPending;

    return (
        <Card variant="outlined">
            <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
                    <Typography variant="caption" color="text.secondary">
                        {reference} · reçu le {new Date(createdAt).toLocaleString('fr-FR')}
                    </Typography>
                    {promotion && <Chip size="small" variant="outlined" label={promotion} />}
                    {badge}
                </Box>
                <TextField
                    fullWidth
                    multiline
                    minRows={3}
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    sx={{ mt: 1 }}
                    label="Texte à publier (corrigez si besoin)"
                />
                {rejecting && (
                    <TextField
                        fullWidth
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                        sx={{ mt: 2 }}
                        label="Motif du refus"
                    />
                )}
                {error && (
                    <Alert severity="error" sx={{ mt: 2 }}>
                        {error}
                    </Alert>
                )}
            </CardContent>
            <CardActions sx={{ justifyContent: 'flex-end', px: 2, pb: 2 }}>
                {!rejecting ? (
                    <>
                        <Button color="error" disabled={busy} onClick={() => setRejecting(true)}>
                            Refuser
                        </Button>
                        <Button
                            variant="contained"
                            disabled={busy}
                            onClick={() => approveMutation.mutate()}
                        >
                            Publier
                        </Button>
                    </>
                ) : (
                    <>
                        <Button disabled={busy} onClick={() => setRejecting(false)}>
                            Annuler
                        </Button>
                        <Button
                            color="error"
                            variant="contained"
                            disabled={busy}
                            onClick={() => rejectMutation.mutate()}
                        >
                            Confirmer le refus
                        </Button>
                    </>
                )}
            </CardActions>
        </Card>
    );
}
