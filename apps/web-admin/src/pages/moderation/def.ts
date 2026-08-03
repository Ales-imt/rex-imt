export const MODERATION = 'moderation';

export const ENDPOINT_MODERATION = `/api/v2/${MODERATION}`;
export const ENDPOINT_MODERATION_VERBATIM = `${ENDPOINT_MODERATION}/verbatim`;

export interface PendingFeedback {
    id: number;
    raw_content: string;
    created_at: string;
    promotion?: string;
}

export interface PendingVerbatim {
    id: string;
    raw_texte: string;
    dimension: string;
    created_at: string;
    promotion?: string;
}

/** Valeur du sélecteur signifiant « pas de filtre ». */
export const PROMOTION_ALL = '';

/** Valeur du sélecteur pour les messages sans promotion renseignée (feedbacks
 *  anonymisés par la purge RGPD, ou cours non rattaché à une promotion). */
export const PROMOTION_NONE = '__none__';

/** Libellés des dimensions d'évaluation, pour donner au modérateur le contexte
 *  de la question à laquelle le verbatim répond. */
export const DIMENSION_LABEL: Record<string, string> = {
    SUPPORTS: 'Supports pédagogiques',
    AMBIANCE: 'Ambiance de classe',
    NPS: 'Recommandation',
};
