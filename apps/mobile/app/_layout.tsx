import { DarkTheme, DefaultTheme, ThemeProvider } from '@react-navigation/native';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';

import { useColorScheme } from '@/hooks/use-color-scheme';
import { AgoraProvider } from '@/hooks/use-agora';
import { setupAxiosInterceptors } from '@/services/api';

export function RootLayout() {
  const colorScheme = useColorScheme();

  useEffect(() => { setupAxiosInterceptors(); }, []);

  return (
    <AgoraProvider>
    <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
      <Stack>
        <Stack.Screen name="index" options={{ headerShown: false }} />
        <Stack.Screen name="chat" options={{ title: 'Chat', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="agora" options={{ title: 'Agora', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="notes" options={{ title: 'Notes', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="programme" options={{ title: 'Programme', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="apropos" options={{ title: 'A propos' }} />
        <Stack.Screen name="evaluation" options={{ title: 'Évaluations', headerBackVisible: false, headerLeft: () => null }} />
      </Stack>
      <StatusBar style="auto" />
    </ThemeProvider>
    </AgoraProvider>
  );
}

export default RootLayout;

