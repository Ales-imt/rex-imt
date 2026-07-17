import { HeaderMenu } from '@/components/header-menu';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { Stack } from 'expo-router';

export default function PointageLayout() {
  const navMenu = useNavMenu('pointage');
  return (
    <>
      <Stack.Screen
        options={{
          title: 'Pointage',
          headerRight: () => <HeaderMenu items={navMenu} />,
        }}
      />
      <Stack>
        <Stack.Screen name="index" options={{ headerShown: false }} />
        <Stack.Screen name="[seanceId]" options={{ title: 'Séance' }} />
      </Stack>
    </>
  );
}
