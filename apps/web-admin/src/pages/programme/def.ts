export const PROGRAMME = "programme"
export const PROGRAMME_WORKFLOW = "programme"
// Clé sessionStorage : dernière période dont le planning a été affiché,
// pour reprendre directement /programme/:periodeId au retour sur l'écran.
export const PROGRAMME_LAST_PERIODE_KEY = "programme.lastPeriodeId"

// evite une dependance circulaire


export interface SalleRef { id: number; name: string; batiment?: string }
export interface IntervenantRef { id: number; firstName?: string; lastName?: string }
export interface GroupeRef { id: number; name: string; option_id: number }


export interface Horaire {
    Lower: string;
    Upper: string;
}

export interface ReservationDetail {
    id: number;
    version: number;
    horaire: Horaire;
    periode_id: number;
    matiere_id?: number | null;
    matiere_name?: string | null;
    matiere_color?: string | null;
    type_cours?: string | null;
    is_distanciel: boolean;
    description?: string | null;
    salles: SalleRef[];
    intervenants: IntervenantRef[];
    groupes: GroupeRef[];
    // Une séance vise SOIT un groupe (groupes non vide), SOIT la promotion
    // entière (promotion_id non nul, groupes vide) : un CM en amphi. Le
    // serveur ne l'aplatit pas en liste de groupes, la distinction remonte
    // intacte et c'est au front d'en tirer les conséquences.
    promotion_id?: number | null;
    promotion_name?: string | null;
    remarque?: string | null;
}

// Réponse de GET /planning/groupes : les groupes de la promotion de la
// période, depuis le référentiel — un groupe sans aucune séance y figure.
export interface GroupeDetail {
    groupe_id: number;
    groupe_name: string;
    taille: number | null;
}

// Ordre des groupes. Un « / » dans le nom désigne une partition de promotion :
// « 1/12 » est le groupe 1 sur 12, « 14/10 » le groupe 10 sur 14 — les deux
// écritures existent, le plus grand nombre est toujours la taille de la
// partition. Clés, dans l'ordre : le texte AVANT la fraction (vide pour
// « 1/12 », qui passe donc en tête ; ce qui suit la fraction — « INFRES 1/2
// SR » — est un libellé d'option et ne compte pas), la taille de partition
// croissante (les demi-promos avant les douzièmes), puis le numéro. Les noms
// sans fraction se comparent sur leur texte, en ordre alphabétique à
// collation numérique (« 3-1 » avant « 10-1 »).
const FRACTION = /(\d+)\s*\/\s*(\d+)/;

function cleGroupe(name: string): { texte: string; partition: number; numero: number } {
    const m = FRACTION.exec(name);
    if (!m) return { texte: name.trim(), partition: 0, numero: 0 };
    const a = Number(m[1]), b = Number(m[2]);
    return { texte: name.slice(0, m.index).trim(), partition: Math.max(a, b), numero: Math.min(a, b) };
}

export function comparerGroupes(a: string, b: string): number {
    const ka = cleGroupe(a), kb = cleGroupe(b);
    return ka.texte.localeCompare(kb.texte, 'fr', { numeric: true })
        || ka.partition - kb.partition
        || ka.numero - kb.numero;
}
