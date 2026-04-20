import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router';


import App from './App.tsx'
import { USER_WORKFLOW } from './pages/user/def.tsx';
import Layout from './layouts/dashboard.tsx';
import { UserIndex } from './pages/user/UserLayout.tsx';
import { createUserRoutes } from './pages/user/routes.tsx';
import { LOGIN } from './pages/login/def.ts';
import SignIn from './pages/login/signin.tsx';

const routes = [
  {
    Component: App,
    children: [
      {
        path: '/',
        Component: Layout,
        children: [
          {
            path: USER_WORKFLOW,
            // element: <RoleGuard roles={[Role.ADMIN]}><UserLayout /></RoleGuard>,
            children: [
              { index: true, Component: UserIndex },
              ...createUserRoutes()
            ]
          }
        ]
      },
      {
        path: '/'+LOGIN,
        Component: SignIn
      }
    ]
  }
]


const router = createBrowserRouter(routes);



createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
)
