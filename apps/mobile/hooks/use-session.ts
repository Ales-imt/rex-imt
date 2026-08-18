import { verifierSession } from '@/services/session';
import { abonnerSession, etatSessionCourant, type EtatSession } from '@/services/tokens';
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
 * 'inconnue', l'état étant illisible hors du navigateur — le HTML statique est
 * donc celui de l'écran d'attente, et non celui du formulaire de connexion.
 */
export function useSession(): EtatSession {
  const etat = useSyncExternalStore(abonnerSession, etatSessionCourant, () => 'inconnue' as EtatSession);
  // Lève l'état 'inconnue' du démarrage : lecture du stockage puis contrôle du
  // jeton auprès du serveur. Idempotent, quel que soit le nombre d'appelants.
  useEffect(() => { void verifierSession(); }, []);
  return etat;
}
