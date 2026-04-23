import { useRouter } from 'expo-router';
import { MenuItem } from '@/components/header-menu';
import { logout } from '@/services/auth';

type Route = 'agora' | 'chat' | 'programme' | 'notes' | 'apropos';

const NAV_ITEMS: { route: Route; icon: string; label: string }[] = [
  { route: 'agora',      icon: '📋', label: 'Agora' },
  { route: 'chat',       icon: '💬', label: 'Chat' },
  { route: 'programme',  icon: '📅', label: 'Programme' },
  { route: 'notes',      icon: '📝', label: 'Notes' },
  { route: 'apropos',    icon: 'ℹ️',  label: 'À propos' },
];

export function useNavMenu(current: Route): MenuItem[] {
  const router = useRouter();

  return [
    ...NAV_ITEMS
      .filter(item => item.route !== current)
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
