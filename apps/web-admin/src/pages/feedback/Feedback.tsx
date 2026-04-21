import { z } from 'zod';
import { Dialog, DialogContent, DialogTitle, IconButton, TextField, Tooltip, Typography } from '@mui/material';
import LockOpenIcon from '@mui/icons-material/LockOpen';
import { useMemo, useState } from 'react';
import { Crud } from '../../services/crud/Crud';
import { createRepository, type CrudProps, type Datasource, type RenderProps, type ViewConfig } from '../../services/crud/def';
import { useRootPath } from '../../services/crud/useRootPath';
import type { MRT_ColumnDef } from 'material-react-table';
import { ENDPOINT_FEEDBACK, FEEDBACK, SENTIMENT_LABELS } from './def';

const feedbackSchema = z.object({
    id: z.number(),
    content: z.string(),
    created_at: z.string().nullish(),
    promotion: z.string().nullish(),
    groupe: z.string().nullish(),
    categorie: z.string().nullish(),
    sous_categorie: z.string().nullish(),
    sentiment: z.string().nullish(),
    urgence: z.number().nullish(),
    resume: z.string().nullish(),
    strongbox: z.string().nullish(),
});

export type Feedback = z.infer<typeof feedbackSchema>;

const FeedbackFields = ({ register, errors }: RenderProps<Feedback>) => (
    <>
        <TextField
            {...register('content')}
            label="Contenu"
            variant="outlined"
            fullWidth
            multiline
            minRows={3}
            disabled
            error={!!errors.content}
            helperText={errors.content?.message}
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('promotion')}
            label="Promotion"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('groupe')}
            label="Groupe"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('categorie')}
            label="Catégorie"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('sous_categorie')}
            label="Sous-catégorie"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('sentiment')}
            label="Sentiment"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('urgence')}
            label="Urgence"
            variant="outlined"
            fullWidth
            disabled
            sx={{ mb: 2 }}
        />
        <TextField
            {...register('resume')}
            label="Résumé IA"
            variant="outlined"
            fullWidth
            multiline
            minRows={2}
            disabled
            sx={{ mb: 2 }}
        />
    </>
);

function StrongboxModal({ value, onClose }: { value: string; onClose: () => void }) {
    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>StrongBox</DialogTitle>
            <DialogContent>
                <Typography
                    component="pre"
                    sx={{ fontFamily: 'monospace', fontSize: '0.85rem', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
                >
                    {value}
                </Typography>
            </DialogContent>
        </Dialog>
    );
}

const baseFeedbackColumns: MRT_ColumnDef<Feedback>[] = [
    { accessorKey: 'id', header: 'ID', size: 60 },
    {
        accessorKey: 'content',
        header: 'Contenu',
        size: 300,
        Cell: ({ cell }) => {
            const val = cell.getValue<string>();
            const truncated = val?.length > 100 ? `${val.slice(0, 100)}…` : val;
            return (
                <Tooltip title={val} placement="top" arrow>
                    <span>{truncated}</span>
                </Tooltip>
            );
        },
    },
    { accessorKey: 'promotion', header: 'Promotion' },
    { accessorKey: 'groupe', header: 'Groupe' },
    { accessorKey: 'categorie', header: 'Catégorie' },
    {
        accessorKey: 'sentiment',
        header: 'Sentiment',
        Cell: ({ cell }) => {
            const val = cell.getValue<string | null>();
            return val ? (SENTIMENT_LABELS[val] ?? val) : '';
        },
    },
    { accessorKey: 'urgence', header: 'Urgence', size: 80 },
    {
        accessorKey: 'created_at',
        header: 'Date',
        size: 120,
        Cell: ({ cell }) => {
            const val = cell.getValue<string | null>();
            if (!val) return '';
            const d = new Date(val);
            return d.toLocaleString('fr-FR', { day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' });
        },
    },
];

function useFeedbackColumns(): { columns: MRT_ColumnDef<Feedback>[]; modal: React.ReactNode } {
    const [strongbox, setStrongbox] = useState<string | null>(null);

    const actionColumn: MRT_ColumnDef<Feedback> = {
        id: 'strongbox-action',
        header: 'StrongBox',
        size: 90,
        enableSorting: false,
        Cell: ({ row }) => {
            const val = row.original.strongbox;
            if (!val) return null;
            return (
                <Tooltip title="Voir StrongBox" placement="top" arrow>
                    <IconButton size="small" onClick={() => setStrongbox(val)}>
                        <LockOpenIcon fontSize="small" />
                    </IconButton>
                </Tooltip>
            );
        },
    };

    const modal = strongbox ? (
        <StrongboxModal value={strongbox} onClose={() => setStrongbox(null)} />
    ) : null;

    return { columns: [...baseFeedbackColumns, actionColumn], modal };
}

export const feedbackViewConfig: ViewConfig<Feedback> = {
    schema: feedbackSchema,
    emptyValue: { id: -1, content: '' },
    columns: baseFeedbackColumns,
    render: FeedbackFields,
};

export const feedbackDatasourceBase = createRepository<Feedback>({
    endpoint: ENDPOINT_FEEDBACK,
    queryKey: [FEEDBACK],
    getId: (data: Feedback) => data.id,
});

export function CrudFeedback({ mode, workflow, isAction, isTopToolbar, renderTopToolbarCustomActions }: CrudProps<Feedback>) {
    const rootPath = useRootPath(mode);
    const { columns, modal } = useFeedbackColumns();

    const datasource = useMemo((): Datasource<Feedback> => ({
        ...feedbackDatasourceBase,
        ...feedbackViewConfig,
        columns,
        title: 'Feedbacks',
        isAction,
        isReadOnly: true,
        isTopToolbar,
        renderTopToolbarCustomActions,
    }), [columns, isAction, isTopToolbar, renderTopToolbarCustomActions]);

    return (
        <>
            <Crud datasource={datasource} mode={mode} workflow={workflow} rootPath={rootPath} />
            {modal}
        </>
    );
}
