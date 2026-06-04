export interface MatiereStatus {
    id: number;
    name: string;
    nb_repondants: number;
    dot_status: 'OK' | 'WARN' | 'NONE';
}

export interface PeriodeTree {
    id: number;
    name: string;
    matieres: MatiereStatus[];
}

export interface PromotionTree {
    id: number;
    name: string;
    periodes: PeriodeTree[];
}

export interface NpsDistribution {
    pct_detracteurs: number;
    pct_passifs: number;
    pct_promoteurs: number;
}

export interface ChipStat {
    chip_id: string;
    libelle: string;
    dimension: string;
    polarite: 'POSITIF' | 'NEGATIF';
    nb: number;
    pct: number;
}

export interface SyntheseIA {
    id: string;
    matiere_id: number;
    semestre: string;
    nb_repondants: number;
    synthese_texte: string;
    statut: 'GENEREE' | 'VALIDEE' | 'ARCHIVEE';
    generated_at: string;
    validated_by: string | null;
    validated_at: string | null;
}

export interface EvalStats {
    nb_repondants: number;
    score_peda: number | null;
    score_clarte: number | null;
    score_contenu: number | null;
    score_difficulte: number | null;
    score_supports: number | null;
    score_ambiance: number | null;
    nps_moyen: number | null;
    pct_charge_lourde: number;
    pct_charge_legere: number;
    pct_format_presentiel: number;
    pct_format_distanciel: number;
    pct_format_hybride: number;
    pct_tendance_baisse: number;
    pct_tendance_stable: number;
    pct_tendance_progres: number;
    nps_distribution: NpsDistribution | null;
    chip_stats: ChipStat[];
    synthese: SyntheseIA | null;
}

export interface Verbatim {
    id: string;
    session_id: string;
    dimension: string;
    texte: string;
    created_at: string;
}

export interface VerbatimsPage {
    data: Verbatim[];
    page: number;
    limit: number;
}

export interface SelectedMatiere {
    id: number;
    name: string;
    periodeName: string;
    promotionName: string;
}

export { EvaluationsPage as Evaluation } from './EvaluationsPage';
