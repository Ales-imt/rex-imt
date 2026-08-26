import axios from 'axios';
import { ensureRefreshed } from './api';
import { API_BASE } from './auth';
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  isExpiringSoon,
  publierSession,
  saveRoles,
} from './tokens';

// Le démarrage attend cette vérification : mieux vaut la déclarer perdue et
// laisser l'application se débrouiller que rester indéfiniment sur l'écran
// d'attente parce qu'un serveur ne répond pas.
const TIMEOUT_MS = 8_000;

let verification: Promise<void> | null = null;

/**
 * Tranche l'état de session au démarrage : 'ouverte' ou 'fermee', jamais
 * 'inconnue' au retour.
 *
 * La seule présence d'un jeton dans le stockage ne prouve rien — il peut être
 * expiré, sa session peut avoir été révoquée côté serveur (déconnexion depuis
 * un autre appareil, compte supprimé). S'en contenter, c'est peindre l'écran
 * d'un élève puis le renvoyer au login sur le premier 401 : exactement le
 * clignotement que l'écran d'attente supprime.
 *
 * Le contrôle passe donc par le serveur, hors intercepteurs (services/api.ts) :
 * l'intercepteur 401 navigue, or à ce stade aucun navigateur n'est encore monté.
 *
 * Idempotente : les appels concurrents (rendus multiples, StrictMode) partagent
 * la même promesse.
 */
export function verifierSession(): Promise<void> {
  verification ??= executerVerification();
  return verification;
}

async function executerVerification(): Promise<void> {
  const acces = await getAccessToken();
  if (!acces) {
    publierSession('fermee');
    return;
  }

  // Jeton d'accès expiré ou sur le point de l'être : le rafraîchissement fait
  // office de contrôle, /auth/refresh vérifiant le jeton de renouvellement en
  // base. Inutile d'y ajouter un appel à /me.
  if (isExpiringSoon(acces)) {
    await rafraichir();
    return;
  }

  await confirmerAupresDuServeur(acces);
}

async function rafraichir(): Promise<void> {
  const renouvellement = await getRefreshToken();
  if (!renouvellement) {
    await clearTokens();
    return;
  }

  try {
    // Passe par le mutex de services/api.ts : si une requête d'écran a déjà
    // déclenché son propre rafraîchissement au même instant, on attend le
    // sien plutôt que d'en tirer un second sur le même refresh token à usage
    // unique. saveTokens (côté doRefresh) publie 'ouverte'.
    await ensureRefreshed();
  } catch (e) {
    // Un refus du serveur (4xx : jeton inconnu, révoqué, compte banni) clôt
    // vraiment la session. Une panne — réseau coupé, serveur en carafe — ne
    // dit rien de sa validité : effacer les jetons déconnecterait un
    // utilisateur valide et lui ferait ressaisir son mot de passe pour une
    // coupure de Wi-Fi.
    if (refusDuServeur(e)) {
      await clearTokens();
      return;
    }
    publierSession('ouverte');
  }
}

async function confirmerAupresDuServeur(acces: string): Promise<void> {
  try {
    const res = await axios.get(`${API_BASE}/me`, {
      headers: { Authorization: `Bearer ${acces}` },
      timeout: TIMEOUT_MS,
    });
    // Les rôles ne sont écrits qu'au login : le démarrage est la seule
    // occasion de les remettre à jour si l'administration les a changés.
    if (Array.isArray(res.data?.roles)) await saveRoles(res.data.roles);
    publierSession('ouverte');
  } catch (e) {
    if (statutHttp(e) === 401 || statutHttp(e) === 403) {
      await clearTokens();
      return;
    }
    // Même raisonnement que pour le rafraîchissement : une panne serveur ne
    // déconnecte pas. Les écrans affichent alors leurs données en cache et
    // gèrent leurs propres erreurs.
    publierSession('ouverte');
  }
}

function statutHttp(e: unknown): number | undefined {
  return axios.isAxiosError(e) ? e.response?.status : undefined;
}

function refusDuServeur(e: unknown): boolean {
  const statut = statutHttp(e);
  return statut !== undefined && statut >= 400 && statut < 500;
}
