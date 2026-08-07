import { EvaluationProvider, useEvaluation } from '@/hooks/use-evaluation';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { Stack, router, usePathname } from 'expo-router';
import { useEffect } from 'react';
import { HeaderMenu } from '@/components/header-menu';

/** Le questionnaire ne peut être rempli qu'en partant de la liste des cours :
 *  elle seule renseigne matiereId (via init). L'état vit en mémoire, donc sur le
 *  web un rechargement (F5) ou une URL d'étape ouverte directement repart d'un
 *  formulaire vierge. Sans ce garde-fou on peut alors remplir les étapes
 *  restantes par-dessus le vide : chaque écran ne valide que ses propres champs,
 *  et c'est l'API qui refuse l'envoi tout à la fin, sans dire pourquoi. */
function GardeQuestionnaire() {
  const { state } = useEvaluation();
  const pathname = usePathname();
  const orphelin = pathname.startsWith('/evaluation/step') && !state.matiereId;

  useEffect(() => {
    if (orphelin) router.replace('/evaluation');
  }, [orphelin]);

  return null;
}

export default function EvaluationLayout() {
  const navMenu = useNavMenu('evaluation');
  return (
    <>

      <EvaluationProvider>
        <GardeQuestionnaire />
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
