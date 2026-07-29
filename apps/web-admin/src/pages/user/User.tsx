import { z } from 'zod';
import { Controller } from 'react-hook-form';
import { createRepository, type CrudProps, type Datasource, type RenderProps, type ViewConfig } from '../../services/crud/def';
import { Box, Checkbox, Chip, FormControl, FormControlLabel, FormGroup, FormHelperText, FormLabel, TextField } from "@mui/material";
import { useMemo } from "react";
import { Crud } from "../../services/crud/Crud";
import { AVAILABLE_ROLES, ENDPOINT_USER, Role, USER } from './def';
import type { MRT_ColumnDef } from 'material-react-table';
import { useRootPath } from '../../services/crud/useRootPath';


const userSchema = z.object({
    id: z.number(),
    keycloak_id: z.string().nullish(),
    version: z.number(),
    surname: z.string().optional(),
    name: z.string().optional(),
    email: z.email("Email invalide").optional().or(z.literal('')),
    password: z.string().optional(),
    roles: z.union([
        z.string().transform((val) => val.split(',').map(r => r.trim()).filter(r => r !== '')),
        z.array(z.string())
    ]).optional(),
    blame: z.boolean().optional(),
});

export type User = z.infer<typeof userSchema>;

const UserFields = ({ register, control, errors, isReadOnly }: RenderProps<User>) => (
    <>
        <TextField
            {...register("surname")}
            label="Prénom"
            variant="outlined"
            fullWidth
            disabled={isReadOnly}
            error={!!errors.surname}
            helperText={errors.surname?.message}
            sx={{ mb: 2 }}
        />
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
            {...register("email")}
            label="Email"
            variant="outlined"
            fullWidth
            disabled={isReadOnly}
            error={!!errors.email}
            helperText={errors.email?.message}
            sx={{ mb: 2 }}
        />
        
        <Controller
            name="roles"
            control={control}
            render={({ field }) => {
                const currentRoles = Array.isArray(field.value) ? field.value : [];
                return (
                    <FormControl component="fieldset" error={!!errors.roles} disabled={isReadOnly} sx={{ mb: 2, width: '100%' }}>
                        <FormLabel component="legend">Rôles</FormLabel>
                        <FormGroup row>
                            {AVAILABLE_ROLES.map((role) => (
                                <FormControlLabel
                                    key={role.id}
                                    control={
                                        <Checkbox
                                            checked={currentRoles.includes(role.id)}
                                            onChange={(e) => {
                                                const checked = e.target.checked;
                                                const newRoles = checked
                                                    ? [...currentRoles, role.id]
                                                    : currentRoles.filter((r: string) => r !== role.id);
                                                field.onChange(newRoles);
                                            }}
                                        />
                                    }
                                    label={role.label}
                                />
                            ))}
                        </FormGroup>
                        {errors.roles && <FormHelperText>{errors.roles.message as string}</FormHelperText>}
                    </FormControl>
                );
            }}
        />

        <Controller
            name="blame"
            control={control}
            render={({ field }) => (
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={!!field.value}
                            onChange={(e) => field.onChange(e.target.checked)}
                            disabled={isReadOnly}
                        />
                    }
                    label="Blamé"
                    sx={{ mb: 2 }}
                />
            )}
        />
    </>
);

export const userColumns: MRT_ColumnDef<User>[] = [
    { accessorKey: 'id', header: 'ID' },
    { accessorKey: 'name', header: 'Nom' },
    { accessorKey: 'surname', header: 'Prénom' },
    { accessorKey: 'email', header: 'Email' },
    {
        accessorKey: 'roles',
        header: 'Rôles',
        Cell: ({ cell }) => {
            const roles = cell.getValue<string[]>();
            const roleColors: Record<string, 'error' | 'warning' | 'primary' | 'info' | 'default'> = {
                [Role.ADMIN]: 'error',
                [Role.GESTIONNAIRE]: 'warning',
                [Role.PROF]: 'primary',
                [Role.ELEVE]: 'info',
            };
            return (
                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                    {Array.isArray(roles) && roles.map(r => (
                        <Chip
                            key={r}
                            label={AVAILABLE_ROLES.find(ar => ar.id === r)?.label || r}
                            color={roleColors[r] ?? 'default'}
                            size="small"
                            variant="filled"
                        />
                    ))}
                </Box>
            );
        }
    },
    {
        accessorKey: 'blame',
        header: 'Blamé',
        Cell: ({ cell }) => (cell.getValue<boolean>() ? 'Oui' : 'Non'),
    },

]

export const userViewConfig: ViewConfig<User> = {
    schema: userSchema,
    emptyValue: { id: -1, version: 0, keycloak_id: "-", roles: [], blame: false },
    columns: userColumns,
    render: UserFields,
};

// Partie statique : à l'extérieur du composant
export const userDatasourceBase = createRepository<User>({
    endpoint: `${ENDPOINT_USER}`,
    queryKey: [USER],
    getId: (data: User) => data.id,
})


export function CrudUser({ mode, workflow, isAction, isTopToolbar, renderTopToolbarCustomActions }: CrudProps<User>) {

    const rootPath = useRootPath(mode);

    const datasource = useMemo((): Datasource<User> => ({
        ...userDatasourceBase,
        ...userViewConfig,
        title: "Utilisateurs",
        isAction,
        isTopToolbar,
        renderTopToolbarCustomActions,
    }), [isAction, isTopToolbar, renderTopToolbarCustomActions]);

    return (
        <Crud datasource={datasource} mode={mode} workflow={workflow} rootPath={rootPath} />
    )
}