import { useLocation } from 'react-router';
import { useMemo } from 'react';
import type { CrudMode } from './def';

// Assurez-vous d'importer votre type et votre fonction
// import { extractRootPath, type CrudMode } from './votreFichierUtilitaire';

export function useRootPath(mode: CrudMode): string {
    const location = useLocation();

    const rootPath = useMemo(() => {
        return extractRootPath(location.pathname, mode);
    }, [location.pathname, mode]);

    return rootPath;
}

function extractRootPath(urlPath: string, mode: CrudMode): string {
    // 1. On ignore les éventuels paramètres d'URL (ex: ?param=1)
    const pathWithoutQuery = urlPath.split('?')[0];

    // 2. On enlève un éventuel slash final pour éviter un segment vide
    const cleanPath = pathWithoutQuery.endsWith('/')
        ? pathWithoutQuery.slice(0, -1)
        : pathWithoutQuery;

    // 3. On découpe le chemin en tableau de segments
    const segments = cleanPath.split('/');

    // 4. On retire les segments finaux selon le mode
    switch (mode) {
        case 'list':
            break
        case 'create':
        case 'show':
            // "rootPath/new" ou "rootPath/10" : on retire 1 segment
            segments.pop();
            break;
        case 'edit':
            // "rootPath/10/edit" : on retire 2 segments ("edit" puis "10")
            segments.pop();
            segments.pop();
            break;
        default:
            throw new Error(`Mode non géré: ${mode}`);
    }

    // 5. On reconstruit l'URL
    return segments.join('/');
}