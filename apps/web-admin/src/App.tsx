import React from 'react';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import { type Authentication, type NavigationItem } from '@toolpad/core/AppProvider';
import PersonIcon from '@mui/icons-material/Person';
import { USER_WORKFLOW } from './pages/user/def';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import SessionContext, { type Session } from './SessionContext';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { ReactRouterAppProvider } from '@toolpad/core/react-router';
import { Outlet, useLocation } from 'react-router';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { getCredentialsProvider } from './pages/login/CredentialProvider';
import { setupAxiosInterceptors } from './services/api';


type NavigationItemWithRoles = NavigationItem & {
  requiredRoles?: string[];
  children?: NavigationItemWithRoles[]
};

const NAVIGATION: NavigationItemWithRoles[] = [
  {
    segment: USER_WORKFLOW,
    title: 'Utilisateur',
    icon: <PersonIcon />,
    // requiredRoles: ['admin'],
  }

]


const BRANDING = {
  title: 'Gestionnaire Scolarite',
};

const queryClient = new QueryClient()


export default function App() {

  const [session, setSession] = React.useState<Session | null>(null);
  const [loading, setLoading] = React.useState(true);
  const location = useLocation();


  console.log("location", location)
  
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
