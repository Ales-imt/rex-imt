import type { ReactNode } from "react";
import { createCrudRoutes, type CrudComponentProps } from "../../services/crud/routes";
import { ANNEE, ANNEE_WORKFLOW } from "./def";
import { CrudAnnee } from "./Annee";
import { Box } from "@mui/material";

export function createAnneeRoutes() {

    const anneePath = ANNEE;

    return [
        createCrudRoutes(anneePath, CustomCrudAnnee),
    ]

}

function CustomCrudAnnee({ mode }: CrudComponentProps) {

    const renderTopToolbar = ({
        defaultActions,
    }: {
        defaultActions: ReactNode;
        isEditMode: boolean;
    }) => (
        <Box sx={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            {defaultActions}
        </Box>
    )
    return <CrudAnnee
        workflow={ANNEE_WORKFLOW}
        mode={mode}
        isAction={true}
        isTopToolbar={true}
        renderTopToolbarCustomActions={renderTopToolbar}
    />;
}
