import * as Crypto from 'expo-crypto';
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import { viderProgrammeCache } from './programmeCache';

const ACCESS_KEY = 'access_token';
const REFRESH_KEY = 'refresh_token';
const PSEUDO_KEY = 'pseudo';
const ROLES_KEY = 'roles';

// expo-secure-store ne fonctionne pas sur web : fallback localStorage
const storage = {
  async getItem(key: string): Promise<string | null> {
    if (Platform.OS === 'web') return localStorage.getItem(key);
    return SecureStore.getItemAsync(key);
  },
  async setItem(key: string, value: string): Promise<void> {
    if (Platform.OS === 'web') { localStorage.setItem(key, value); return; }
    return SecureStore.setItemAsync(key, value);
  },
  async deleteItem(key: string): Promise<void> {
    if (Platform.OS === 'web') { localStorage.removeItem(key); return; }
    return SecureStore.deleteItemAsync(key);
  },
};

// ─── État de session observable ─────────────────────────────────────────────
//
// La garde de navigation (cf. app/_layout.tsx) doit savoir s'il existe une
// session AVANT qu'un écran protégé ne soit monté. Ni localStorage ni
// SecureStore ne notifient de leurs changements : l'état est donc tenu ici, au
// seul endroit qui écrit les jetons.
//
// 'inconnue' n'est pas 'fermee' : c'est « pas encore tranché ». La distinction
// est vitale — c'est l'état du démarrage, celui pendant lequel le RootLayout
// n'affiche que l'écran d'attente, et le confondre avec une absence de session
// éjecterait vers le login des utilisateurs parfaitement connectés.
//
// Cet état n'est levé que par verifierSession() (services/session.ts) : la
// présence d'un jeton dans le stockage ne suffit pas à conclure 'ouverte', un
// jeton pouvant être expiré ou sa session révoquée côté serveur.
export type EtatSession = 'inconnue' | 'ouverte' | 'fermee';

let etatSession: EtatSession = 'inconnue';
const abonnesSession = new Set<() => void>();

export function publierSession(etat: EtatSession): void {
  if (etat === etatSession) return;
  etatSession = etat;
  abonnesSession.forEach(cb => cb());
}

/** État courant, lisible pendant le rendu (cf. hooks/use-session.ts). */
export function etatSessionCourant(): EtatSession {
  return etatSession;
}

export function abonnerSession(cb: () => void): () => void {
  abonnesSession.add(cb);
  return () => { abonnesSession.delete(cb); };
}

export function generateUUID(): string {
  const bytes = Crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

// ─── Liste de mots français simples du quotidien ─────────────────────────────

const WORDS = [
  'arbre', 'bateau', 'bière', 'bois', 'bougie', 'branche', 'brique', 'brume',
  'bureau', 'canard', 'carnet', 'cerise', 'chaise', 'champ', 'chapeau', 'chat',
  'cheval', 'chien', 'ciel', 'citron', 'clé', 'colline', 'couleur', 'crayon',
  'drap', 'dune', 'eau', 'écharpe', 'école', 'écran', 'été', 'étoile',
  'fenêtre', 'feuille', 'fleur', 'forêt', 'four', 'fraise', 'fromage', 'fruit',
  'gare', 'gazon', 'genou', 'glace', 'globe', 'gomme', 'grain', 'graine',
  'herbe', 'hiver', 'horloge', 'image', 'jardin', 'journal', 'jouet', 'jupe',
  'lac', 'lampe', 'lapin', 'lettre', 'lièvre', 'livre', 'loup', 'lumière',
  'lune', 'main', 'maison', 'matin', 'mer', 'miel', 'miroir', 'montagne',
  'mouton', 'mur', 'nuit', 'nuage', 'oiseau', 'orange', 'oreille', 'os',
  'pain', 'papier', 'parc', 'pied', 'pierre', 'plage', 'plante', 'pluie',
  'poisson', 'pomme', 'pont', 'porte', 'poule', 'prairie', 'printemps', 'puits',
  'queue', 'racine', 'radio', 'renard', 'repas', 'rivière', 'robe', 'rocher',
  'rose', 'route', 'ruban', 'sable', 'saison', 'sel', 'sentier', 'soleil',
  'soupe', 'source', 'stylo', 'sucre', 'table', 'tasse', 'terre', 'toit',
  'torche', 'tour', 'train', 'vallée', 'vase', 'vent', 'vitre', 'voile',
  'abricot', 'accord', 'agenda', 'aiguille', 'aile', 'algue', 'amande', 'ami',
  'ancre', 'ange', 'année', 'ardoise', 'armoire', 'automne', 'avion', 'bague',
  'balcon', 'balle', 'banane', 'banc', 'bandeau', 'bâton', 'beurre', 'biberon',
  'bocal', 'botte', 'bouche', 'bouée', 'bourgeon', 'boussole', 'brindille',
  'broche', 'brouette', 'bulle', 'cabane', 'caillou', 'calendrier', 'canapé',
  'canard', 'caramel', 'carotte', 'carreau', 'cartable', 'casque', 'castor',
  'cerf', 'chou', 'cible', 'cire', 'cloche', 'cœur', 'coin', 'colombe',
  'compas', 'corde', 'couteau', 'couvercle', 'crabe', 'crème', 'crochet',
  'cueillette', 'dauphin', 'dessin', 'digue', 'dossier', 'élan', 'éponge',
  'escalier', 'étang', 'facette', 'falaise', 'fameux', 'farine', 'faucon',
  'figue', 'fil', 'flûte', 'fond', 'fontaine', 'foulard', 'frêne', 'fuseau',
];

export function generatePseudo(): string {
  const arr = [...WORDS];
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Crypto.getRandomValues(new Uint32Array(1))[0] % (i + 1);
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr.slice(0, 4).join(' . ');
}

export async function savePseudo(pseudo: string): Promise<void> {
  await storage.setItem(PSEUDO_KEY, pseudo);
}

export async function getPseudo(): Promise<string | null> {
  return storage.getItem(PSEUDO_KEY);
}

export async function getOrCreatePseudo(): Promise<string> {
  let pseudo = await storage.getItem(PSEUDO_KEY);
  if (!pseudo) {
    pseudo = generatePseudo();
    await storage.setItem(PSEUDO_KEY, pseudo);
  }
  return pseudo;
}

export async function saveTokens(access: string, refresh: string): Promise<void> {
  await storage.setItem(ACCESS_KEY, access);
  await storage.setItem(REFRESH_KEY, refresh);
  publierSession('ouverte');
}

export async function getAccessToken(): Promise<string | null> {
  return storage.getItem(ACCESS_KEY);
}

export async function getRefreshToken(): Promise<string | null> {
  return storage.getItem(REFRESH_KEY);
}

// Rôles reçus au login, utilisés uniquement pour masquer/afficher des entrées
// de menu — le contrôle d'accès réel est fait côté backend.
export async function saveRoles(roles: string[]): Promise<void> {
  await storage.setItem(ROLES_KEY, JSON.stringify(roles));
}

export async function getRoles(): Promise<string[]> {
  const raw = await storage.getItem(ROLES_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export async function clearTokens(): Promise<void> {
  await storage.deleteItem(ACCESS_KEY);
  await storage.deleteItem(REFRESH_KEY);
  await storage.deleteItem(PSEUDO_KEY);
  await storage.deleteItem(ROLES_KEY);
  // Point de passage unique de la fin de session (déconnexion explicite comme
  // échec de rafraîchissement) : le cache du planning part avec elle, sans quoi
  // le compte suivant sur l'appareil verrait le planning du précédent.
  viderProgrammeCache();
  publierSession('fermee');
}

export function isExpiringSoon(token: string, thresholdSec = 30): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return Date.now() / 1000 > (payload.exp ?? 0) - thresholdSec;
  } catch {
    return true;
  }
}
