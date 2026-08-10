// DTO des excuses, alignés sur backend/admin/pkg/justification.
//
// Une excuse ne porte NI motif, NI justificatif, NI commentaire libre : la
// raison de l'absence se traite hors logiciel, par e-mail entre l'étudiant et
// le gestionnaire. C'est ce qui garantit qu'aucune donnée de santé n'entre dans
// le système. N'ajoutez pas de champ de ce type ici.

export type StatutJustification = 'ACTIVE' | 'ANNULEE' | 'REMPLACEE';

export interface Justification {
    id: number;
    user_id: number;
    name: string;
    surname: string;
    starts_at: string;
    ends_at: string;
    nb_seances: number;
    statut: StatutJustification;
    created_at: string;
    created_by: number;
    created_by_name: string;
    created_by_surname: string;
    revoked_at: string | null;
    replaces_id: number | null;
    replaced_by_id: number | null;
}

export interface SeanceApercu {
    id: number;
    starts_at: string;
    ends_at: string;
    matiere: string;
    salle: string;
    statut: 'PRESENT' | 'RETARD' | 'ABSENT';
    /** La séance tombe dans la plage saisie. */
    dans_plage: boolean;
    /** La séance est couverte par la justification en cours de modification. */
    deja_couverte: boolean;
}

export interface Chevauchement {
    id: number;
    starts_at: string;
    ends_at: string;
}

export interface ApercuJustification {
    seances: SeanceApercu[];
    chevauchements: Chevauchement[];
}
