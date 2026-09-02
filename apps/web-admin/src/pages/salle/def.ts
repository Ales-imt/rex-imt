import dayjs, { Dayjs } from 'dayjs';

export const SALLE_DISPO_WORKFLOW      = 'salle-dispo';
export const SALLE_OCCUPATION_WORKFLOW = 'salle-occupation';
export const SALLE_SEMAINE_WORKFLOW    = 'salle-semaine';
export const ENDPOINT_SALLES           = '/api/v2/planning/salles';

// Clés sessionStorage de l'écran Semaine, partagées : Occupation écrit la
// semaine avant de naviguer, pour que le clic ouvre la même semaine.
export const SALLE_SEMAINE_LUNDI_KEY = 'salle.semaine.lundi';
export const SALLE_SEMAINE_SALLE_KEY = 'salle.semaine.salle';

// Réponse de GET /salles/occupation. Elle part des salles (LEFT JOIN) : une
// salle jamais réservée sort avec heures: 0 — c'est LA source de la liste
// complète des salles, capacité et type compris.
export interface OccupationSalle {
    salle_id:   number;
    salle_name: string;
    capacite:   number | null; // null = non renseignée en amont, pas « 0 place »
    type:       string | null;
    nb_seances: number;
    heures:     number;
}

// Réponse de GET /salles/creneaux. INNER JOIN sur salle : une séance sans
// salle_id n'y figure pas — elle n'occupe aucun local.
export interface CreneauSalle {
    salle_id:       number;
    salle_name:     string;
    seance_id:      number;
    starts_at:      string; // RFC 3339 avec décalage
    ends_at:        string;
    matiere_name:   string;
    prof:           string | null;
    groupe_name:    string | null;
    promotion_name: string | null;
}

export function fmtHeure(iso: string): string {
    return dayjs(iso).format('HH:mm');
}

// « TP Réseaux (2A-G3) · M. DUPONT » — le groupe prime sur la promotion,
// il est plus précis.
export function libelleOccupant(c: CreneauSalle): string {
    const cible = c.groupe_name || c.promotion_name;
    let s = c.matiere_name;
    if (cible) s += ` (${cible})`;
    if (c.prof) s += ` · ${c.prof}`;
    return s;
}

export interface StatutSalle {
    // Plusieurs occupants simultanés = double réservation en amont : on les
    // affiche tous plutôt que d'en choisir un arbitrairement.
    occupants: CreneauSalle[];
    // Pour une salle libre : le prochain créneau de la journée, ou null.
    prochain:  CreneauSalle | null;
}

// Statut d'une salle à l'instant t, sur les créneaux de SA journée.
// On compare des instants parsés, jamais des chaînes HH:mm : la comparaison
// de chaînes marcherait par accident tant que serveur et navigateur partagent
// le fuseau, et casserait en silence dès qu'un navigateur n'y est pas.
export function statutSalle(creneaux: CreneauSalle[], t: Dayjs): StatutSalle {
    const occupants: CreneauSalle[] = [];
    let prochain: CreneauSalle | null = null;
    for (const c of creneaux) {
        const debut = dayjs(c.starts_at);
        const fin = dayjs(c.ends_at);
        if (!debut.isAfter(t) && fin.isAfter(t)) {
            occupants.push(c);
        } else if (debut.isAfter(t) && (!prochain || debut.isBefore(dayjs(prochain.starts_at)))) {
            prochain = c;
        }
    }
    return { occupants, prochain };
}
