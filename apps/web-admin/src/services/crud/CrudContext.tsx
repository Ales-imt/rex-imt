import { createContext, useContext } from 'react';

export interface CrudContextType {
    rootPath: string;
    workflow: string;
}

const CrudContext = createContext<CrudContextType | undefined>(undefined);

export const useCrudContext = () => {
    const context = useContext(CrudContext);
    if (!context) {
        throw new Error('useCrudContext must be used within a CrudProvider');
    }
    return context;
};

export const CrudProvider = CrudContext.Provider;