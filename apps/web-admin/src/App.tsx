import React from 'react';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import { type Authentication, type NavigationItem } from '@toolpad/core/AppProvider';
import PersonIcon from '@mui/icons-material/Person';
import FeedbackIcon from '@mui/icons-material/Feedback';
import BarChartIcon from '@mui/icons-material/BarChart';
import ForumIcon from '@mui/icons-material/Forum';
import { Role, USER_WORKFLOW } from './pages/user/def';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import SessionContext, { type Session } from './SessionContext';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { ReactRouterAppProvider } from '@toolpad/core/react-router';
import { Outlet } from 'react-router';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { getCredentialsProvider } from './pages/login/CredentialProvider';
import { setupAxiosInterceptors } from './services/api';
import { FEEDBACK_WORKFLOW } from './pages/feedback/def';
import { ANALYSE } from './pages/analyse/def';
import { DISCUSSION } from './pages/discussion/def';
import { EVALUATION_WORKFLOW } from './pages/evaluation/def';
import { PRESENCE_WORKFLOW, PRESENCE_WITNESS_WORKFLOW } from './pages/presence/def';
import HowToRegIcon from '@mui/icons-material/HowToReg';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import { ANNEE_WORKFLOW } from './pages/annee/def';
import EventIcon from '@mui/icons-material/Event';


type NavigationItemWithRoles = NavigationItem & {
  requiredRoles?: string[];
  children?: NavigationItemWithRoles[]
};

const NAVIGATION: NavigationItemWithRoles[] = [
  {
    title: 'Admin',
    icon: <PersonIcon />,
    requiredRoles: [Role.ADMIN],
    children: [
      {
        segment: USER_WORKFLOW,
        title: 'Utilisateur',
        icon: <PersonIcon />,
        requiredRoles: [Role.ADMIN],
      },
      {
        segment: FEEDBACK_WORKFLOW,
        title: 'Feedback',
        icon: <FeedbackIcon />,
        requiredRoles: [Role.ADMIN],
      },
      {
        segment: ANNEE_WORKFLOW,
        title: 'Année',
        icon: <EventIcon />,
        requiredRoles: [Role.ADMIN],
      },
      {
        segment: PRESENCE_WITNESS_WORKFLOW,
        title: 'Audit présences',
        icon: <FactCheckIcon />,
        requiredRoles: [Role.ADMIN, Role.GESTIONNAIRE],
      },
    ],
  },
  {
    segment: ANALYSE,
    title: 'Analyse',
    icon: <BarChartIcon />,
    requiredRoles: [Role.ADMIN, Role.GESTIONNAIRE],
  },
  {
    segment: DISCUSSION,
    title: 'Discussion',
    icon: <ForumIcon />,
    requiredRoles: [Role.ADMIN, Role.GESTIONNAIRE],
  },
  {
    segment: EVALUATION_WORKFLOW,
    title: 'Évaluations',
    icon: <BarChartIcon />,
    requiredRoles: [Role.ADMIN, Role.GESTIONNAIRE],
  },
  {
    segment: PRESENCE_WORKFLOW,
    title: 'Présence',
    icon: <HowToRegIcon />,
    requiredRoles: [Role.ADMIN, Role.GESTIONNAIRE],
  },

]


const BRANDING = {
  title: 'Gestionnaire Scolarite',
};

const queryClient = new QueryClient()


export default function App() {

  const [session, setSession] = React.useState<Session | null>(null);
  const [loading, setLoading] = React.useState(true);

  const sessionContextValue = React.useMemo(
    () => ({
      session,
      setSession,
    }),
    [session],
  );

  const filteredNavigation = React.useMemo(
    () => (
      filterNavigationByRoles(NAVIGATION, session?.user?.roles)
    ),
    [session],
  );

  React.useEffect(() => {

    // S'abonner aux notifications d'initialisation
    const unsubscribe = getCredentialsProvider().subscribe((newSession) => {

      // Mettre à jour la session avec les infos utilisateur
      setSession(newSession);
    });

    // si on a une session , la restaure.
    setSession(getCredentialsProvider().getSession())
    setupAxiosInterceptors()
    setLoading(false)

    // Nettoyer l'abonnement au démontage du composant
    return () => {
      unsubscribe();
    };
  }, [setSession]);

  const AUTHENTICATION: Authentication = {
    signIn: () => { },
    signOut: () => { getCredentialsProvider().signOut() },
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="100vh">
        <CircularProgress />
      </Box>
    )
  }


  return (
    <SessionContext.Provider value={sessionContextValue}>
      <QueryClientProvider client={queryClient}>
        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="fr">
          <ReactRouterAppProvider
            navigation={filteredNavigation}
            branding={BRANDING}
            session={session}
            authentication={AUTHENTICATION}
            localeText={{
              accountSignInLabel: "Connexion",
              accountSignOutLabel: "Deconexion",
            }}
          >
            <Outlet />
          </ReactRouterAppProvider>
        </LocalizationProvider>
      </QueryClientProvider>
    </SessionContext.Provider >
  )

}

const filterNavigationByRoles = (
  navigation: NavigationItemWithRoles[],
  userRoles: string[] | undefined
): NavigationItemWithRoles[] => {
  return navigation
    .filter(item => !item.requiredRoles || item.requiredRoles.some(role => userRoles?.includes(role)))
    .map(item => ({
      ...item,
      children: item.children
        ? filterNavigationByRoles(item.children, userRoles)
        : undefined,
    }))
    .filter(item => !item.children || item.children.length > 0);
};
