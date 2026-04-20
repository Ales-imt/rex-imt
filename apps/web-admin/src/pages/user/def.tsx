
export const USER = "user"
export const USER_WORKFLOW = "user_workflow"

export const ENDPOINT_USER = `/api/v2/${USER}`


export const Role = {
    ADMIN: 'ADMIN',
    ELEVE: 'ELEVE',
} as const;

export const AVAILABLE_ROLES = [
    { id: Role.ADMIN, label: 'Administrateur' },
    { id: Role.ELEVE, label: 'Élève' },
];