// Formatage des excuses : dates en français lisible, jamais en ISO — ces
// libellés sont lus par un gestionnaire, pas par une machine.

import type { StatutJustification } from './types';

/** « lundi 2 mars 08:00 » */
export function formatInstantFR(iso: string): string {
    const d = new Date(iso);
    const jour = d.toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' });
    const heure = d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
    return `${jour} ${heure}`;
}

/** « du lundi 2 mars 08:00 au mercredi 4 mars 18:00 » */
export function formatPlageFR(debut: string, fin: string): string {
    return `du ${formatInstantFR(debut)} au ${formatInstantFR(fin)}`;
}

export function formatHeure(iso: string): string {
    return new Date(iso).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
}

/** « mar. 5 mai · 09:00–11:00 · B204 » */
export function formatSeanceFR(s: { starts_at: string; ends_at: string; salle?: string }): string {
    const d = new Date(s.starts_at);
    const jour = d.toLocaleDateString('fr-FR', { weekday: 'short', day: 'numeric', month: 'long' });
    const parts = [jour, `${formatHeure(s.starts_at)}–${formatHeure(s.ends_at)}`];
    if (s.salle) parts.push(s.salle);
    return parts.join(' · ');
}

/** Valeur d'un <input type="datetime-local"> pour un instant donné. */
export function toInputLocal(iso: string): string {
    const d = new Date(iso);
    const p = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

/** Instant absolu correspondant à une saisie datetime-local (heure du poste).
 *  L'API accepte aussi une heure locale nue, qu'elle interprète en heure de
 *  Paris ; on lui envoie néanmoins un instant daté, pour que ce que le
 *  gestionnaire lit dans le champ soit exactement ce qui est enregistré. */
export function fromInputLocal(valeur: string): string | null {
    if (!valeur) return null;
    const d = new Date(valeur);
    return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

export function dureeHeures(seances: { starts_at: string; ends_at: string }[]): number {
    const ms = seances.reduce(
        (total, s) => total + (new Date(s.ends_at).getTime() - new Date(s.starts_at).getTime()),
        0,
    );
    return Math.round((ms / 3_600_000) * 10) / 10;
}

export function dureeJours(debut: string, fin: string): number {
    const ms = new Date(fin).getTime() - new Date(debut).getTime();
    return Math.max(0, Math.ceil(ms / 86_400_000));
}

export const STATUT_JUSTIFICATION: Record<StatutJustification, { label: string; color: 'success' | 'default' | 'info' }> = {
    ACTIVE: { label: 'Active', color: 'success' },
    ANNULEE: { label: 'Annulée', color: 'default' },
    REMPLACEE: { label: 'Remplacée', color: 'info' },
};

/** « Compte supprimé » quand l'identité du saisissant a été anonymisée : la FK
 *  ON DELETE RESTRICT interdit sa suppression, mais pas son anonymisation. */
export function nomSaisissant(nom: string, prenom: string): string {
    const complet = `${nom} ${prenom}`.trim();
    return complet === '' ? 'Compte supprimé' : complet;
}
