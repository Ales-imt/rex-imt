import { useState, useEffect, useRef } from 'react';
import { apiInstance } from '../../services/api';
import type { EvalStats, PromotionTree, SyntheseIA, VerbatimsPage } from './Evaluation';
import { ENDPOINT_EVALUATION } from './def';

interface AsyncState<T> {
    data: T | null;
    loading: boolean;
    error: string | null;
    depsKey: string;
}

function useAsync<T>(fetchFn: () => Promise<T>, deps: unknown[]) {
    const depsKey = JSON.stringify(deps);
    const epochRef = useRef(0);

    const [state, setState] = useState<AsyncState<T>>({
        data: null,
        loading: true,
        error: null,
        depsKey,
    });

    // If deps changed since last completed fetch, signal loading during this render
    const effective = state.depsKey !== depsKey
        ? { data: state.data, loading: true, error: null }
        : state;

    useEffect(() => {
        const epoch = ++epochRef.current;
        const key = JSON.stringify(deps);
        fetchFn()
            .then(data => {
                if (epochRef.current === epoch)
                    setState({ data, loading: false, error: null, depsKey: key });
            })
            .catch((e: unknown) => {
                if (epochRef.current === epoch)
                    setState(s => ({ ...s, loading: false, error: (e as Error)?.message ?? 'Erreur', depsKey: key }));
            });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);

    return effective;
}

export function useAnnees() {
    return useAsync<number[]>(
        () => apiInstance.get<number[]>(`${ENDPOINT_EVALUATION}/annees`).then(r => r.data),
        [],
    );
}

export function usePromotionTree(annee: number | null) {
    return useAsync<PromotionTree[]>(
        () => annee != null
            ? apiInstance.get<PromotionTree[]>(`${ENDPOINT_EVALUATION}/promotions?annee=${annee}`).then(r => r.data)
            : Promise.resolve([]),
        [annee],
    );
}

export function useMatiereStats(matiereId: number | null) {
    return useAsync<EvalStats>(
        () => matiereId != null
            ? apiInstance.get<EvalStats>(`${ENDPOINT_EVALUATION}/matieres/${matiereId}/stats`).then(r => r.data)
            : Promise.resolve(null as unknown as EvalStats),
        [matiereId],
    );
}

export function useGenerateSynthese(matiereId: number) {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const generate = async (): Promise<SyntheseIA | null> => {
        setLoading(true);
        setError(null);
        try {
            const { data } = await apiInstance.post<SyntheseIA>(
                `${ENDPOINT_EVALUATION}/matieres/${matiereId}/synthese`,
            );
            return data;
        } catch (e: unknown) {
            setError((e as Error)?.message ?? 'Erreur lors de la génération');
            return null;
        } finally {
            setLoading(false);
        }
    };

    return { generate, loading, error };
}

export function useValidateSynthese() {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const validate = async (syntheseId: string, validatedBy: string): Promise<boolean> => {
        setLoading(true);
        setError(null);
        try {
            await apiInstance.patch(`${ENDPOINT_EVALUATION}/syntheses/${syntheseId}/statut`, {
                statut: 'VALIDEE',
                validated_by: validatedBy,
            });
            return true;
        } catch (e: unknown) {
            setError((e as Error)?.message ?? 'Erreur lors de la validation');
            return false;
        } finally {
            setLoading(false);
        }
    };

    return { validate, loading, error };
}

export function useVerbatims(matiereId: number | null, dimension?: string, page = 1) {
    return useAsync<VerbatimsPage>(
        () => {
            if (matiereId == null) return Promise.resolve({ data: [], page: 1, limit: 10 });
            const params = new URLSearchParams({ page: String(page), limit: '10' });
            if (dimension) params.set('dimension', dimension);
            return apiInstance
                .get<VerbatimsPage>(`${ENDPOINT_EVALUATION}/matieres/${matiereId}/verbatims?${params}`)
                .then(r => r.data);
        },
        [matiereId, dimension, page],
    );
}
