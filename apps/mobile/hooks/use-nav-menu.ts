import { useRouter } from 'expo-router';
import { useEffect, useState } from 'react';
import { MenuItem } from '@/components/header-menu';
import { logout } from '@/services/auth';
import { getRoles } from '@/services/tokens';

type Route = 'agora' | 'chat' | 'programme' | 'presence' | 'pointage' | 'notes' | 'evaluation' | 'apropos';

// `roles` : entrée visible uniquement si l'utilisateur a l'un de ces rôles.
// Simple masquage d'UI — le contrôle d'accès réel est fait côté backend.
const NAV_ITEMS: { route: Route; icon: string; label: string; roles?: string[] }[] = [
  { route: 'agora',      icon: '📋', label: 'Agora' },
  { route: 'chat',       icon: '💬', label: 'Chat' },
  { route: 'programme',  icon: '📅', label: 'Programme', roles: ['PROF','ELEVE'] },
  { route: 'presence',   icon: '✅', label: 'Présence' , roles: ['ELEVE']},
  { route: 'pointage',   icon: '🧑‍🏫', label: 'Pointage', roles: ['PROF', 'GESTIONNAIRE'] },
  { route: 'notes',      icon: '📝', label: 'Notes' ,roles: ['ELEVE']},
  { route: 'evaluation', icon: '⭐', label: 'Évaluations' ,roles: ['ELEVE']},
  { route: 'apropos',    icon: 'ℹ️',  label: 'À propos' },
];

export function useNavMenu(current: Route): MenuItem[] {
  const router = useRouter();
  const [roles, setRoles] = useState<string[]>([]);

  useEffect(() => {
    let cancelled = false;
    getRoles().then(r => { if (!cancelled) setRoles(r); }).catch(() => { });
    return () => { cancelled = true; };
  }, []);

  return [
    ...NAV_ITEMS
      .filter(item => item.route !== current)
      .filter(item => !item.roles || item.roles.some(r => roles.includes(r)))
      .map(item => ({
        icon: item.icon,
        label: item.label,
        onPress: () => router.push(`/${item.route}`),
      })),
    {
      icon: '🚪',
      label: 'Se déconnecter',
      onPress: () => { logout().then(() => router.replace('/')).catch(() => router.replace('/')); },
      danger: true,
    },
  ];
}
