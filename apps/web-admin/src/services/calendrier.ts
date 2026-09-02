import { Dayjs } from 'dayjs';

// Amplitudes d'ouverture proposées : une convention d'établissement, elle se
// choisit ici et jamais en SQL — la figer côté base interdirait de la faire
// varier sans redéploiement. Elle sert trois fois : dénominateur du taux
// d'Occupation (heures), bornes de la grille de Semaine d'une salle et bornes
// de la grille Groupes du programme (debut/fin/samedi). Une seule définition,
// sinon les écrans divergent sans que personne ne s'en aperçoive.
export const AMPLITUDES = [
    { cle: '8-18x5', label: '8h–18h · lun-ven', heures: 50, debut: '08:00:00', fin: '18:00:00', samedi: false },
    { cle: '8-20x5', label: '8h–20h · lun-ven', heures: 60, debut: '08:00:00', fin: '20:00:00', samedi: false },
    { cle: '8-18x6', label: '8h–18h · lun-sam', heures: 60, debut: '08:00:00', fin: '18:00:00', samedi: true  },
] as const;

export type Amplitude = (typeof AMPLITUDES)[number];

export function lundiDe(d: Dayjs): string {
    return d.subtract((d.day() + 6) % 7, 'day').format('YYYY-MM-DD');
}

export function labelSemaine(lundi: string): string {
    const d = new Date(lundi + 'T12:00:00');
    return 'Semaine du ' + d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' });
}
