import { z } from 'zod';
import { createRepository, type CrudProps, type Datasource, type RenderProps, type ViewConfig } from '../../services/crud/def';
import { TextField } from "@mui/material";
import { useMemo } from "react";
import { Crud } from "../../services/crud/Crud";
import { ENDPOINT_ANNEE, ANNEE } from './def';
import type { MRT_ColumnDef } from 'material-react-table';
import { useRootPath } from '../../services/crud/useRootPath';


const anneeSchema = z.object({
    id: z.number(),
    name: z.string().min(1, "Nom obligatoire"),
    debut: z.string().min(1, "Date de début obligatoire"),
    fin: z.string().min(1, "Date de fin obligatoire"),
});

export type Annee = z.infer<typeof anneeSchema>;

const AnneeFields = ({ register, errors, isReadOnly }: RenderProps<Annee>) => (
    <>
        <TextField
            {...register("name")}
            label="Nom"
            variant="outlined"
            fullWidth
            disabled={isReadOnly}
            error={!!errors.name}
            helperText={errors.name?.message}
            sx={{ mb: 2 }}
        />
        <TextField
            {...register("debut")}
            label="Début"
            type="date"
            variant="outlined"
            fullWidth
            disabled={isReadOnly}
            error={!!errors.debut}
            helperText={errors.debut?.message}
            slotProps={{ inputLabel: { shrink: true } }}
            sx={{ mb: 2 }}
        />
        <TextField
            {...register("fin")}
            label="Fin"
            type="date"
            variant="outlined"
            fullWidth
            disabled={isReadOnly}
            error={!!errors.fin}
            helperText={errors.fin?.message}
            slotProps={{ inputLabel: { shrink: true } }}
            sx={{ mb: 2 }}
        />
    </>
);

export const anneeColumns: MRT_ColumnDef<Annee>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'name', header: 'Nom' },
    { accessorKey: 'debut', header: 'Début' },
    { accessorKey: 'fin', header: 'Fin' },
]

export const anneeViewConfig: ViewConfig<Annee> = {
    schema: anneeSchema,
    emptyValue: { id: -1, name: '', debut: '', fin: '' },
    columns: anneeColumns,
    render: AnneeFields,
};

// Partie statique : à l'extérieur du composant
export const anneeDatasourceBase = createRepository<Annee>({
    endpoint: `${ENDPOINT_ANNEE}`,
    queryKey: [ANNEE],
    getId: (data: Annee) => data.id,
})


export function CrudAnnee({ mode, workflow, isAction, isTopToolbar, renderTopToolbarCustomActions }: CrudProps<Annee>) {

    const rootPath = useRootPath(mode);

    const datasource = useMemo((): Datasource<Annee> => ({
        ...anneeDatasourceBase,
        ...anneeViewConfig,
        title: "Années",
        isAction,
        isTopToolbar,
        renderTopToolbarCustomActions,
    }), [isAction, isTopToolbar, renderTopToolbarCustomActions]);

    return (
        <Crud datasource={datasource} mode={mode} workflow={workflow} rootPath={rootPath} />
    )
}
