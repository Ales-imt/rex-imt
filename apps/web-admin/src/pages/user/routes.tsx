import type { ReactNode } from "react";
import { createCrudRoutes, type CrudComponentProps } from "../../services/crud/routes";
import { USER, USER_WORKFLOW } from "./def";
import { CrudUser } from "./User";
import { Box } from "@mui/material";

export function createUserRoutes() {

    const userPath = USER;

    return [
        createCrudRoutes(userPath, CustomCrudUser),
    ]

}

function CustomCrudUser({ mode }: CrudComponentProps) {

    const renderTopToolbar = ({
        defaultActions,
    }: {
        defaultActions: ReactNode;
        isEditMode: boolean;
    }) => (
        <Box sx={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
            {defaultActions}
            {/* Import : composant React autonome avec ses propres hooks */}
        </Box>
    )
    return <CrudUser
        workflow={USER_WORKFLOW}
        mode={mode}
        isAction={true}
        isTopToolbar={true}
        renderTopToolbarCustomActions={renderTopToolbar}
    />;
}