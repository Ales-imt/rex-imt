import Stack from '@mui/material/Stack';

import { Outlet, useNavigate } from 'react-router';
import { DashboardLayout } from '@toolpad/core/DashboardLayout';
import { Account } from '@toolpad/core/Account';

import { useSession } from '../SessionContext';
import { createTheme, ThemeProvider, useColorScheme, type Theme } from '@mui/material/styles';
import { NotificationsProvider } from '@toolpad/core/useNotifications';
import { useEffect } from 'react';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#6366F1', dark: '#4F46E5', light: '#818CF8' },
    secondary: { main: '#22C55E' },
    error: { main: '#EF4444' },
    warning: { main: '#F97316' },
    info: { main: '#0EA5E9' },
    success: { main: '#22C55E' },
  },
  shape: { borderRadius: 10 },
  typography: { fontFamily: 'system-ui, sans-serif' },
});

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#818CF8', dark: '#6366F1', light: '#A5B4FC' },
    secondary: { main: '#4ADE80' },
    error: { main: '#F87171' },
    warning: { main: '#FB923C' },
    info: { main: '#38BDF8' },
    success: { main: '#4ADE80' },
  },
  shape: { borderRadius: 10 },
});

function CustomActions() {
  return (
    <Stack direction="row" alignItems="center">
      <Account
        slotProps={{
          preview: { slotProps: { avatarIconButton: { sx: { border: '0' } } } },
        }}
      />
    </Stack>
  );
}

export default function Layout() {
  const { session } = useSession()
  const navigate = useNavigate();
  const { mode } = useColorScheme()

  useEffect(() => {
    if (!session) navigate('/login')
  }, [session, navigate])

  //  Le return null évite de rendre le layout pendant la redirection.
  if (!session) return null;

  var theme: Theme
  if (mode == undefined || mode == 'system') {
    theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? darkTheme : lightTheme
  } else {
    theme = mode === 'dark' ? darkTheme : lightTheme
  }

  return (
    <NotificationsProvider
      slotProps={{
        snackbar: {
          anchorOrigin: { vertical: 'top', horizontal: 'center' },
        },
      }}
    >
      <ThemeProvider theme={theme}>
        <DashboardLayout
          defaultSidebarCollapsed
          slots={{
            toolbarActions: CustomActions
          }}
          sx={{
            background: theme.palette.background.default,
            backgroundColor: theme.palette.background.default
          }}
        >
          <Outlet />
        </DashboardLayout>
      </ThemeProvider>

    </NotificationsProvider>
  )
}