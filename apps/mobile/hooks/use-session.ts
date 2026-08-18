import { abonnerSession, etatSessionCourant, resoudreSession, type EtatSession } from '@/services/tokens';
import { useEffect, useSyncExternalStore } from 'react';

/**
 * État de session, réactif au login comme à la déconnexion.
 *
 * useSyncExternalStore parce que la source de vérité est hors de React (le
 * stockage), et que la valeur doit être disponible PENDANT le rendu : c'est ce
 * qui permet à la garde de navigation d'écarter un écran protégé avant qu'il
 * ne soit monté, au lieu de le démonter après coup.
 *
 * Le troisième argument est le snapshot du rendu statique : il y renvoie
 * 'inconnue', l'état étant illisible hors du navigateur.
 */
export function useSession(): EtatSession {
  const etat = useSyncExternalStore(abonnerSession, etatSessionCourant, () => 'inconnue' as EtatSession);
  // Sur natif, la première lecture est asynchrone ; sur web, no-op.
  useEffect(() => { void resoudreSession(); }, []);
  return etat;
}
