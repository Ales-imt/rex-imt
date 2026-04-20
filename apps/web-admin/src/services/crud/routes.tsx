import type { ComponentType } from "react"
import type { CrudMode } from "./def";

const NEW = "new"
const EDIT = ":id/edit"
const SHOW = ":id"

export interface CrudComponentProps {
  mode: CrudMode
}

// Helper pour créer les routes CRUD avec injection du contexte
export function createCrudRoutes(path: string, Component: ComponentType<CrudComponentProps>) {
  return {
    path,
    children: [
      {
        index: true,
        Component: () => <Component mode="list" />
      },
      {
        path: NEW,
        Component: () => <Component mode="create" />
      },
      {
        path: SHOW,
        Component: () => <Component mode="show" />
      },
      {
        path: EDIT,
        Component: () => <Component mode="edit" />
      }
    ]
  }
}

export function createRoute(path: string, Component: ComponentType) {
  return {
    path,
    index: true,
    Component: () => <Component />
  }
}