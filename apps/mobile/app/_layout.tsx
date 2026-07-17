import { DarkTheme, DefaultTheme, ThemeProvider } from '@react-navigation/native';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { Platform } from 'react-native';

import { useColorScheme } from '@/hooks/use-color-scheme';
import { AgoraProvider } from '@/hooks/use-agora';
import { setupAxiosInterceptors } from '@/services/api';

export function RootLayout() {
  const colorScheme = useColorScheme();

  useEffect(() => { setupAxiosInterceptors(); }, []);

  useEffect(() => {
    if (Platform.OS !== 'web') return;

    // Configurer le viewport pour forcer le redimensionnement du layout par le clavier sous Firefox/Android
    let viewportMeta = document.querySelector('meta[name="viewport"]') as HTMLMetaElement | null;
    if (viewportMeta) {
      let content = viewportMeta.content;
      if (!content.includes('interactive-widget')) {
        viewportMeta.content = `${content}, interactive-widget=resizes-content`;
      }
    } else {
      viewportMeta = document.createElement('meta');
      viewportMeta.name = 'viewport';
      viewportMeta.content = 'width=device-width, initial-scale=1, shrink-to-fit=no, interactive-widget=resizes-content';
      document.head.appendChild(viewportMeta);
    }

    let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null;
    if (!meta) {
      meta = document.createElement('meta');
      meta.name = 'theme-color';
      document.head.appendChild(meta);
    }
    const color = colorScheme === 'dark' ? '#0d1117' : '#f0f4f8';
    meta.content = color;

    let schemeMeta = document.querySelector('meta[name="color-scheme"]') as HTMLMetaElement | null;
    if (!schemeMeta) {
      schemeMeta = document.createElement('meta');
      schemeMeta.name = 'color-scheme';
      document.head.appendChild(schemeMeta);
    }
    schemeMeta.content = colorScheme === 'dark' ? 'dark' : 'light';

    document.documentElement.style.backgroundColor = color;
    document.body.style.backgroundColor = color;
  }, [colorScheme]);

  return (
    <AgoraProvider>
    <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
      <Stack>
        <Stack.Screen name="index" options={{ headerShown: false }} />
        <Stack.Screen name="chat" options={{ title: 'Chat', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="agora" options={{ title: 'Agora', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="notes" options={{ title: 'Notes', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="programme" options={{ title: 'Programme', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="presence" options={{ title: 'Présence', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="pointage" options={{ title: 'Pointage', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="apropos" options={{ title: 'A propos' }} />
        <Stack.Screen name="apropos-first-login" options={{ title: 'À propos', headerBackVisible: false, headerLeft: () => null }} />
        <Stack.Screen name="pseudo-setup" options={{ headerShown: false }} />
        <Stack.Screen name="evaluation" options={{ title: 'Évaluations', headerBackVisible: false, headerLeft: () => null }} />
      </Stack>
      <StatusBar style="auto" />
    </ThemeProvider>
    </AgoraProvider>
  );
}

export default RootLayout;

