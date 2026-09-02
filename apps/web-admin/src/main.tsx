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
import { PRESENCE_WORKFLOW, PRESENCE_WITNESS_WORKFLOW } from './pages/presence/def.ts';
import { Presence } from './pages/presence/Presence.tsx';
import { VerifyWitness } from './pages/presence/VerifyWitness.tsx';
import { JUSTIFICATION_WORKFLOW } from './pages/justification/def.ts';
import { Justifications } from './pages/justification/Justifications.tsx';
import { ANNEE_WORKFLOW } from './pages/annee/def.ts';
import { AnneeIndex } from './pages/annee/AnneeLayout.tsx';
import { createAnneeRoutes } from './pages/annee/routes.tsx';
import { PROGRAMME_WORKFLOW } from './pages/programme/def.ts';
import { ProgrammeSelect, ProgrammeIndex } from './pages/programme/ProgrammeSelect.tsx';
import { Planning } from './pages/programme/Planning.tsx';
import { Groupes } from './pages/programme/Groupes.tsx';
import { BULLETTIN_WORKFLOW } from './pages/bullettin/def.ts';
import { Bullettin } from './pages/bullettin/Bullettin.tsx';
import { MODERATION } from './pages/moderation/def.ts';
import { Moderation } from './pages/moderation/Moderation.tsx';
import { SALLE_DISPO_WORKFLOW, SALLE_OCCUPATION_WORKFLOW, SALLE_SEMAINE_WORKFLOW } from './pages/salle/def.ts';
import { Disponibilite } from './pages/salle/Disponibilite.tsx';
import { Occupation } from './pages/salle/Occupation.tsx';
import { Semaine } from './pages/salle/Semaine.tsx';

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
            path: ANNEE_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN]}><Outlet /></RoleGuard>,
            children: [
              { index: true, Component: AnneeIndex },
              ...createAnneeRoutes()
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
            path: MODERATION,
            element: <RoleGuard roles={[Role.ADMIN, Role.MODERATEUR]}><Moderation /></RoleGuard>,
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
          },
          {
            path: JUSTIFICATION_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Justifications /></RoleGuard>,
          },
          {
            path: PRESENCE_WITNESS_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><VerifyWitness /></RoleGuard>,
          },
          {
            path: PROGRAMME_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Outlet /></RoleGuard>,
            children: [
              { index: true, Component: ProgrammeIndex },
              { path: 'select', Component: ProgrammeSelect },
              { path: ':periodeId', Component: Planning },
              { path: ':periodeId/groupes', Component: Groupes },
            ]
          },
          {
            path: SALLE_DISPO_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Disponibilite /></RoleGuard>,
          },
          {
            path: SALLE_OCCUPATION_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Occupation /></RoleGuard>,
          },
          {
            path: SALLE_SEMAINE_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Semaine /></RoleGuard>,
          },
          {
            path: `${SALLE_SEMAINE_WORKFLOW}/:salleId`,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Semaine /></RoleGuard>,
          },
          {
            path: BULLETTIN_WORKFLOW,
            element: <RoleGuard roles={[Role.ADMIN, Role.GESTIONNAIRE]}><Bullettin /></RoleGuard>,
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
