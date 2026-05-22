export type Mode = 'prive' | 'public';

export interface Message {
  id: number;
  student: string;
  initials: string;
  text: string;
  time: string;
  isTeacher?: boolean;
}

export interface Group {
  id: number;
  emoji: string;
  title: string;
  preview: string;
  studentCount: number;
  badgeColor: string;
  timestamp: string;
  messages: Message[];
}

export interface PostIt {
  id: number;
  bgColor: string;
  author: string;
  text: string;
  tag: string;
  timestamp: string;
  rotation: number;
  x: number;
  y: number;
  isTeacher?: boolean;
}


export const POST_ITS: PostIt[] = [
  {
    id: 1,
    bgColor: '#fef9c3',
    author: 'LUCAS B.',
    text: "Vraiment du mal avec la récursivité, surtout les arbres. Un cours de rattrapage serait top !",
    tag: 'Algo',
    timestamp: '14:32',
    rotation: -1.5,
    x: 55,
    y: 80,
  },
  {
    id: 2,
    bgColor: '#fce7f3',
    author: 'AMINA K.',
    text: "Les cas de base en récursion sont flous. Besoin d'exemples visuels.",
    tag: 'Algo',
    timestamp: '14:33',
    rotation: 1.2,
    x: 270,
    y: 55,
  },
  {
    id: 3,
    bgColor: '#dbeafe',
    author: 'EMMA L.',
    text: "Quelles protections pour le TP chimie ? Je veux être sûre avant d'arriver.",
    tag: 'Chimie',
    timestamp: '14:20',
    rotation: -0.8,
    x: 490,
    y: 90,
  },
  {
    id: 4,
    bgColor: '#dcfce7',
    author: 'LÉA P.',
    text: "Le format de l'examen final n'est toujours pas clair. QCM ou rédactionnel ?",
    tag: 'Examens',
    timestamp: '14:05',
    rotation: 2,
    x: 700,
    y: 65,
  },
  {
    id: 5,
    bgColor: '#fed7aa',
    author: 'TOM V.',
    text: "Les exercices sont bien plus complexes que le cours. Besoin de TD supplémentaires.",
    tag: 'Algo',
    timestamp: '14:35',
    rotation: -1,
    x: 160,
    y: 270,
  },
  {
    id: 6,
    bgColor: '#fef9c3',
    author: 'JULES F.',
    text: "useState ou useReducer ? Aucun critère clair dans le cours. Quelle règle ?",
    tag: 'React',
    timestamp: '13:20',
    rotation: 1.8,
    x: 410,
    y: 280,
  },
  {
    id: 7,
    bgColor: '#dbeafe',
    author: 'YOUNES B.',
    text: "La notation de Fubini pour les intégrales triples. Pouvez-vous refaire un exemple ?",
    tag: 'Maths',
    timestamp: '12:30',
    rotation: -2.2,
    x: 630,
    y: 260,
  },
  {
    id: 8,
    bgColor: '#fce7f3',
    author: 'MARINE C.',
    text: "Mon composant React part en boucle infinie à cause de useEffect. SOS !",
    tag: 'React',
    timestamp: '13:22',
    rotation: 0.9,
    x: 80,
    y: 450,
  },
  {
    id: 9,
    bgColor: '#fffbeb',
    author: 'PROFESSEUR',
    text: "Les annales 2023–2025 sont disponibles sur Moodle. Bon courage pour les révisions !",
    tag: 'Infos',
    timestamp: '14:15',
    rotation: -0.5,
    x: 460,
    y: 445,
    isTeacher: true,
  },
];

export const CONNECTIONS: [number, number][] = [
  [0, 1],
  [0, 4],
  [1, 2],
  [3, 6],
  [4, 5],
  [5, 8],
  [6, 7],
];

export const POSTIT_COLORS = [
  { label: 'Jaune', value: '#fef9c3' },
  { label: 'Rose', value: '#fce7f3' },
  { label: 'Bleu', value: '#dbeafe' },
  { label: 'Vert', value: '#dcfce7' },
  { label: 'Orange', value: '#fed7aa' },
];

const POSTIT_DEFAULT_COLOR = '#fef9c3';

const URGENCE_COLORS: Record<number, string> = {
  1: '#dcfce7', // vert  — faible
  2: '#dbeafe', // bleu  — bas
  3: '#fef9c3', // jaune — moyen
  4: '#fed7aa', // orange — élevé
  5: '#fce7f3', // rose  — critique
};

export function colorFromUrgence(urgence?: number | null): string {
  if (urgence == null) return POSTIT_DEFAULT_COLOR;
  return URGENCE_COLORS[urgence] ?? POSTIT_DEFAULT_COLOR;
}
