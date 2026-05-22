import { EvaluationProvider } from '@/hooks/use-evaluation';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { Stack } from 'expo-router';
import { HeaderMenu } from '@/components/header-menu';

export default function EvaluationLayout() {
  const navMenu = useNavMenu('evaluation');
  return (
    <>

      <EvaluationProvider>
        <Stack.Screen
          options={{
            title: 'Mes évaluations',
            headerRight: () => <HeaderMenu items={navMenu} />,
          }}
        />
        <Stack
          screenOptions={{
            headerStyle: { backgroundColor: '#0F172A' },
            headerTintColor: '#fff',
            headerTitleStyle: { color: '#fff', fontWeight: '600' },
          }}
        >
          <Stack.Screen name="index" options={{ headerShown: false }} />
          <Stack.Screen name="eval-detail" />
          <Stack.Screen name="step0" />
          <Stack.Screen name="step1" />
          <Stack.Screen name="step2" />
          <Stack.Screen name="step3" />
          <Stack.Screen name="step4" />
          <Stack.Screen name="step5" />
          <Stack.Screen name="step6" />
        </Stack>
      </EvaluationProvider>
    </>
  );
}
