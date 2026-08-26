import { File, Paths } from 'expo-file-system';
import { Platform } from 'react-native';

// Cache local de la dernière semaine de planning affichée.
//
// Son but n'est PAS d'économiser des requêtes : /programme interroge webdfd en
// direct et doit continuer à le faire. Il sert à supprimer la grille vide au
// lancement — le contenu connu s'affiche immédiatement pendant que la requête
// part, et il est remplacé dès la réponse (stale-while-revalidate).
//
// L'API de expo-file-system est synchrone : la lecture peut donc alimenter
// l'état initial de l'écran, sans le vide d'un premier rendu asynchrone.
//
// Contenu : des horaires de cours, salles et noms d'enseignants — rien de
// sensible, mais rattaché à UNE personne. Le cache est donc effacé avec la
// session (cf. clearTokens), sans quoi le compte suivant sur le même appareil
// verrait le planning du précédent.

const NOM_FICHIER = 'programme-cache.json';
const CLE_WEB = 'programme.cache';

// Incrémenter à tout changement de forme d'ApiCours : un cache écrit par une
// version antérieure serait sinon relu comme s'il était valide.
const VERSION = 2;

type Cache = {
    version: number;
    weekStart: string;
    cours: unknown[];
};

// Contrat synchrone utilisé sur les plateformes natives.
//
// expo-file-system expose bien exists/textSync/write/delete, mais le tsconfig
// du projet place « .web » dans moduleSuffixes : TypeScript résout donc en
// premier la variante web du module natif, dont la classe est un stub vide
// (`declare class FileSystemFile { constructor(); }`). Les membres natifs
// disparaissent alors du type de File.
//
// Ce n'est pas qu'un ennui de typage : ce stub confirme que l'API fichier n'a
// pas d'implémentation sur web — d'où la branche localStorage ci-dessous, qui
// est le seul chemin emprunté par le navigateur.
type FichierSync = {
    exists: boolean;
    textSync(): string;
    write(contenu: string): void;
    delete(): void;
};

function fichier(): FichierSync {
    return new File(Paths.cache, NOM_FICHIER) as unknown as FichierSync;
}

function lireBrut(): string | null {
    if (Platform.OS === 'web') return localStorage.getItem(CLE_WEB);
    const f = fichier();
    return f.exists ? f.textSync() : null;
}

/** Dernière semaine mise en cache, ou null si absente, illisible ou périmée. */
export function lireProgrammeCache(): { weekStart: string; cours: unknown[] } | null {
    try {
        const brut = lireBrut();
        if (!brut) return null;
        const parsed = JSON.parse(brut) as Cache;
        if (parsed?.version !== VERSION || typeof parsed.weekStart !== 'string' || !Array.isArray(parsed.cours)) {
            return null;
        }
        return { weekStart: parsed.weekStart, cours: parsed.cours };
    } catch {
        // Un cache illisible n'est jamais une erreur pour l'appelant : il
        // affichera simplement une grille vide le temps de la requête.
        return null;
    }
}

export function ecrireProgrammeCache(weekStart: string, cours: unknown[]): void {
    const contenu = JSON.stringify({ version: VERSION, weekStart, cours } satisfies Cache);
    try {
        if (Platform.OS === 'web') {
            localStorage.setItem(CLE_WEB, contenu);
            return;
        }
        fichier().write(contenu);
    } catch {
        // Écriture best-effort : échouer ici ne doit rien casser à l'affichage.
    }
}

export function viderProgrammeCache(): void {
    try {
        if (Platform.OS === 'web') {
            localStorage.removeItem(CLE_WEB);
            return;
        }
        const f = fichier();
        if (f.exists) f.delete();
    } catch {
        // idem
    }
}
