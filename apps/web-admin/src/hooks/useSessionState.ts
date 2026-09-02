import { useState } from 'react';

// État persisté en sessionStorage : les filtres et la période observée
// survivent à une navigation vers un autre écran et retour.
export function useSessionState<T extends string>(key: string, defaut: T): [T, (v: T) => void] {
    const [valeur, setValeur] = useState<T>(() => (sessionStorage.getItem(key) as T | null) ?? defaut);
    const set = (v: T) => {
        setValeur(v);
        sessionStorage.setItem(key, v);
    };
    return [valeur, set];
}
