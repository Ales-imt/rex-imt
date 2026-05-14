import Stack from '@mui/material/Stack';

import { Outlet, useNavigate } from 'react-router';
import { DashboardLayout } from '@toolpad/core/DashboardLayout';
import { Account } from '@toolpad/core/Account';

import { useSession } from '../SessionContext';
import { createTheme, ThemeProvider, useColorScheme, type Theme } from '@mui/material/styles';
import { NotificationsProvider } from '@toolpad/core/useNotifications';
import { useEffect } from 'react';

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
  },
});

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
  },
})

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