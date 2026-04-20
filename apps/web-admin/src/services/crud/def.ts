import type { QueryKey } from '@tanstack/react-query';
import type { Control, DefaultValues, FieldErrors, FieldValues, UseFormGetValues, UseFormRegister, UseFormSetValue } from 'react-hook-form';
import type { JSX, ReactNode } from 'react';
import type { MRT_ColumnDef, MRT_Row, MRT_TableInstance } from 'material-react-table';
import { apiInstance } from '../api';

export type CrudMode = 'create' | 'show' | 'edit' | 'list';

export interface CrudProps<D extends FieldValues> {
    mode: CrudMode
    workflow: string;
    isAction: boolean
    isReadOnly?: boolean
    renderRowActions?: (props: { row: MRT_Row<D>, table: MRT_TableInstance<D>, defaultActions: ReactNode, isEditMode: boolean }) => ReactNode
    isTopToolbar: boolean
    renderTopToolbarCustomActions?: (props: { table: MRT_TableInstance<D>, defaultActions: React.ReactNode, isEditMode: boolean }) => React.ReactNode
}

export interface RenderProps<D extends FieldValues> {
    register: UseFormRegister<D>
    control: Control<D>
    errors: FieldErrors<D>
    isReadOnly: boolean
    getValues: UseFormGetValues<D>
    setValue: UseFormSetValue<D>
}

export interface Repository<D extends FieldValues> {
    queryKey: QueryKey;
    getId: (data: D) => number;
    update: (data: D) => Promise<D>;
    create: (data: D) => Promise<D>;
    fetch: (ids: string | undefined) => Promise<D>;
    fetchAll: () => Promise<D[]>;
    delete: (id: number[]) => Promise<{ success: boolean }>;
}

export interface ViewConfig<D extends FieldValues> {
    schema: any
    emptyValue: DefaultValues<D>;
    columns: MRT_ColumnDef<D>[];
    render: (props: RenderProps<D>) => JSX.Element;
}

export interface Datasource<D extends FieldValues> extends Repository<D>, ViewConfig<D> {
    title: string
    first?: boolean
    isAction: boolean
    isReadOnly?: boolean
    renderRowActions?: (props: { row: MRT_Row<D>, table: MRT_TableInstance<D>, defaultActions: ReactNode, isEditMode: boolean }) => ReactNode
    isTopToolbar: boolean
    renderTopToolbarCustomActions?: (props: { table: MRT_TableInstance<D>, defaultActions: React.ReactNode, isEditMode: boolean }) => React.ReactNode
}

interface RepositoryConfig<D extends FieldValues> {
    endpoint: string;
    queryParams?: string;
    queryKey: QueryKey;
    getId: (data: D) => number;
}

export function createRepository<T extends FieldValues>(
    config: RepositoryConfig<T>
): Repository<T> {
    const endpoint = config.endpoint
    return {
        queryKey: config.queryKey,
        getId: config.getId,

        // Récupérer tous les éléments
        fetchAll: async () => {
            try {
                const url = config.queryParams ? `${endpoint}${config.queryParams}` : endpoint;
                const rep = await apiInstance.get<T[]>(url);
                return rep.data
            } catch (error: any) {
                throw handleAxiosError(error);
            }
        },

        // Récupérer un élément par ID
        fetch: async (id) => {
            try {
                const rep = await apiInstance.get<T>(`${endpoint}/${id}`);
                return rep.data
            } catch (error: any) {
                throw handleAxiosError(error);
            }

        },

        // Créer
        create: async (data) => {
            try {
                const rep = await apiInstance.post<T>(endpoint, data);
                return rep.data
            } catch (error: any) {
                throw handleAxiosError(error);
            }
        },

        // Mettre à jour
        update: async (data) => {
            try {
                const id = config.getId(data);
                const rep = await apiInstance.put<T>(`${endpoint}/${id}`, data);
                return rep.data
            } catch (error: any) {
                throw handleAxiosError(error);
            }
        },

        // Supprimer (Bulk)
        delete: async (ids: (string | number)[]) => {
            try {
                const rep = await apiInstance.delete(`${endpoint}/bulk`, { data: { ids } });
                return rep.data
            } catch (error: any) {
                throw handleAxiosError(error);
            }
        },
    };
}

/**
 * Helper pour transformer les erreurs Axios en votre format ApiError
 * compatible avec votre backend Go
 */
export function handleAxiosError(error: any) {
    if (error.response) {
        // L'erreur vient du serveur (400, 401, 409...)
        return new ApiError(error.response.status, error.response.data);
    }
    return new Error("Erreur réseau ou serveur injoignable");
}


export class ApiError extends Error {
    payload: ErrorResponse | undefined;
    status: number;
    constructor(status: number, payload?: ErrorResponse) {
        super('API Error');
        this.status = status;
        this.payload = payload;
    }

    static async fromResponse(response: Response): Promise<ApiError> {
        const errorBody = await response.json() as ErrorResponse;
        return new ApiError(response.status, errorBody);
    }
}

export interface ErrorResponse {
    code: number
    message: string
    details?: any
}

export const ERROR_CODE = {
    NO_INFORMATION: 0,
    VALIDATION_ERROR: 1,
    OPTIMISTIC_LOCKING_FAILURE: 2
};
