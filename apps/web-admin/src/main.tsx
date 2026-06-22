import { StrictMode, useContext } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider, Outlet } from 'react-router';

import App from './App.tsx'
import { Role, USER_WORKFLOW } from './pages/user/def.tsx';
import Layout from './layouts/dashboard.tsx';
import SessionContext from './SessionContext.ts';
import { UserIndex } from './pages/user/UserLayout.tsx';
import { createUserRoutes } from './pages/user/routes.tsx';
import { LOGIN } from './pages/login/def.ts';
import SignIn from './pages/login/signin.tsx';
import { FEEDBACK_WORKFLOW } from './pages/feedback/def.ts';
import { FeedbackIndex } from './pages/feedback/FeedbackLayout.tsx';
import { createFeedbackRoutes } from './pages/feedback/routes.tsx';
import { EVALUATION_WORKFLOW } from './pages/evaluation/def.ts';
import { ANALYSE } from './pages/analyse/def.ts';
import { FeedbackDashboard } from './pages/analyse/FeedbackDashboard.tsx';
import { DISCUSSION } from './pages/discussion/def.ts';
import { DiscussionPanel } from './pages/discussion/DiscussionPanel.tsx';
import { Evaluation } from './pages/evaluation/Evaluation.tsx';
import { PRESENCE_WORKFLOW } from './pages/presence/def.ts';
import { Presence } from './pages/presence/Presence.tsx';

const RoleGuard = ({ children, roles }: { children: React.ReactNode, roles: string[] }) => {
  const { session } = useContext(SessionContext);

  if (!session?.user) {
    return null;
  }

  const userRoles = session.user.roles || [];
  const hasRole = roles.some(r => userRoles.includes(r));

  if (!hasRole) {
    return <div>Accès non autorisé</div>;
  }

  return <>{children}</>;
};

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
            element: <RoleGuard roles={[Role.ADMIN]}><Outlet /></RoleGuard>,
            children: [
              { index: true, Component: UserIndex },
              ...createUserRoutes()
            ]
          },
          {
            path: FEEDBACK_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Outlet /></RoleGuard>,
            children: [
              { index: true, Component: FeedbackIndex },
              ...createFeedbackRoutes()
            ]
          },
          {
            path: ANALYSE,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><FeedbackDashboard /></RoleGuard>,
          },
          {
            path: DISCUSSION,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><DiscussionPanel /></RoleGuard>,
          },
          {
            path: EVALUATION_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Evaluation /></RoleGuard>,
          },
          {
            path: PRESENCE_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Presence /></RoleGuard>,
          }
        ]
      },
      {
        path: LOGIN,
        Component: SignIn
      },
    ]
  }
]


const router = createBrowserRouter(routes);


createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
)
