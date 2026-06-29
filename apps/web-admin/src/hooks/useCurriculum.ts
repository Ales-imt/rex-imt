import { useState, useEffect, useRef } from 'react';
import { apiInstance } from '../services/api';

export interface MatiereStatus {
    id: number;
    name: string;
    nb_repondants: number;
    dot_status: 'OK' | 'WARN' | 'NONE';
}

export interface PeriodeTree {
    id: number;
    name: string;
    matieres: MatiereStatus[];
}

export interface PromotionTree {
    id: number;
    name: string;
    periodes: PeriodeTree[];
}

const ENDPOINT = '/api/v2/curriculum';

interface AsyncState<T> {
    data: T | null;
    loading: boolean;
}

function useAsync<T>(fetchFn: () => Promise<T>, deps: unknown[]): AsyncState<T> {
    const depsKey = JSON.stringify(deps);
    const epochRef = useRef(0);

    const [state, setState] = useState<AsyncState<T> & { depsKey: string }>({
        data: null,
        loading: true,
        depsKey,
    });

    const effective: AsyncState<T> = state.depsKey !== depsKey
        ? { data: state.data, loading: true }
        : { data: state.data, loading: state.loading };

    useEffect(() => {
        const epoch = ++epochRef.current;
        const key = JSON.stringify(deps);
        fetchFn().then(data => {
            if (epochRef.current === epoch)
                setState({ data, loading: false, depsKey: key });
        }).catch(() => {
            if (epochRef.current === epoch)
                setState(s => ({ ...s, loading: false, depsKey: key }));
        });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);

    return effective;
}

export function useAnnees() {
    return useAsync<number[]>(
        () => apiInstance.get<number[]>(`${ENDPOINT}/annees`).then(r => r.data),
        [],
    );
}

export function usePromotionTree(annee: number | null) {
    return useAsync<PromotionTree[]>(
        () => annee != null
            ? apiInstance.get<PromotionTree[]>(`${ENDPOINT}/promotions?annee=${annee}`).then(r => r.data)
            : Promise.resolve([]),
        [annee],
    );
}

export function useAnneePromo() {
    const { data: annees, loading: anneesLoading } = useAnnees();
    const [selectedAnnee, setSelectedAnnee] = useState<number | null>(null);
    const annee = selectedAnnee ?? (annees?.[0] ?? null);

    const { data: tree, loading: treeLoading } = usePromotionTree(annee);

    const [selectedPromoId, setSelectedPromoId] = useState<number | null>(null);
    useEffect(() => { setSelectedPromoId(tree?.[0]?.id ?? null); }, [tree]);

    const selectedPromo = tree?.find(p => p.id === selectedPromoId) ?? null;

    return {
        annees,
        anneesLoading,
        annee,
        setSelectedAnnee,
        tree,
        treeLoading,
        selectedPromoId,
        setSelectedPromoId,
        selectedPromo,
    };
}
