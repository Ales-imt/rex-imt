
export const USER = "user"
export const USER_WORKFLOW = "user_workflow"

export const ENDPOINT_USER = `/api/v2/${USER}`


export const Role = {
    ADMIN: 'ADMIN',
    GESTIONNAIRE: 'GESTIONNAIRE',
    PROF: 'PROF',
    ELEVE: 'ELEVE',
    MODERATEUR: 'MODERATEUR',
} as const;

export const AVAILABLE_ROLES = [
    { id: Role.ADMIN, label: 'Administrateur' },
    { id: Role.GESTIONNAIRE, label: 'Gestionnaire' },
    { id: Role.PROF, label: 'Professeur' },
    { id: Role.ELEVE, label: 'Élève' },
    { id: Role.MODERATEUR, label: 'Modérateur' },
];