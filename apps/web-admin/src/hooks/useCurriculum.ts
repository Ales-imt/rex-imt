import { useState, useEffect, useRef, useCallback } from 'react';
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

// ─── Persistance du choix (année / promo / période / matière) ───────────────────
// Le choix est mémorisé en sessionStorage, restitué après un changement de page
// ou un rechargement dans le même onglet. Les clés sont préfixées par `scope`
// (nom de l'écran) : chaque écran conserve donc son choix indépendamment des
// autres (évaluation, présence, programme…).

function readNum(key: string): number | null {
    const raw = sessionStorage.getItem(key);
    if (raw == null || raw === '') return null;
    const n = Number(raw);
    return Number.isNaN(n) ? null : n;
}

function writeNum(key: string, v: number | null) {
    if (v == null) sessionStorage.removeItem(key);
    else sessionStorage.setItem(key, String(v));
}

export function useAnneePromo(scope: string) {
    const kAnnee   = `${scope}.annee`;
    const kPromo   = `${scope}.promoId`;
    const kPeriode = `${scope}.periodeId`;
    const kMatiere = `${scope}.matiereId`;

    const { data: annees, loading: anneesLoading } = useAnnees();

    const [selectedAnnee, setSelectedAnneeState] = useState<number | null>(() => readNum(kAnnee));
    const setSelectedAnnee = useCallback((a: number | null) => {
        setSelectedAnneeState(a);
        writeNum(kAnnee, a);
    }, [kAnnee]);
    const annee = selectedAnnee ?? (annees?.[0] ?? null);

    const { data: tree, loading: treeLoading } = usePromotionTree(annee);

    // ── Promotion ────────────────────────────────────────────────────────────
    const [selectedPromoId, setSelectedPromoIdState] = useState<number | null>(() => readNum(kPromo));
    const setSelectedPromoId = useCallback((id: number | null) => {
        setSelectedPromoIdState(id);
        writeNum(kPromo, id);
    }, [kPromo]);

    // À chaque nouvel arbre : on conserve la promo mémorisée si elle existe
    // encore, sinon on retombe sur la première. On ignore un arbre vide (encore
    // en cours de chargement, ou année pas encore résolue) pour ne pas écraser
    // le choix mémorisé avant l'arrivée des vraies données.
    useEffect(() => {
        if (!tree || tree.length === 0) return;
        setSelectedPromoIdState(prev => {
            const next = prev != null && tree.some(p => p.id === prev)
                ? prev
                : (tree[0]?.id ?? null);
            writeNum(kPromo, next);
            return next;
        });
    }, [tree, kPromo]);

    const selectedPromo = tree?.find(p => p.id === selectedPromoId) ?? null;

    // ── Période ──────────────────────────────────────────────────────────────
    const [selectedPeriodeId, setSelectedPeriodeIdState] = useState<number | null>(() => readNum(kPeriode));
    const setSelectedPeriodeId = useCallback((id: number | null) => {
        setSelectedPeriodeIdState(id);
        writeNum(kPeriode, id);
    }, [kPeriode]);

    useEffect(() => {
        if (!selectedPromo) return;
        setSelectedPeriodeIdState(prev => {
            const next = prev != null && selectedPromo.periodes.some(p => p.id === prev)
                ? prev
                : (selectedPromo.periodes?.[0]?.id ?? null);
            writeNum(kPeriode, next);
            return next;
        });
    }, [selectedPromo, kPeriode]);

    const selectedPeriode = selectedPromo?.periodes?.find(p => p.id === selectedPeriodeId) ?? null;

    // ── Matière ──────────────────────────────────────────────────────────────
    const [selectedMatiereId, setSelectedMatiereIdState] = useState<number | null>(() => readNum(kMatiere));
    const setSelectedMatiereId = useCallback((id: number | null) => {
        setSelectedMatiereIdState(id);
        writeNum(kMatiere, id);
    }, [kMatiere]);

    // On valide la matière mémorisée contre la période courante ; si elle n'y
    // figure pas (changement de période/promo), on réinitialise. Tant que
    // l'arbre est vide (chargement), on ne touche à rien pour ne pas perdre le choix.
    useEffect(() => {
        if (!tree || tree.length === 0) return;
        setSelectedMatiereIdState(prev => {
            if (prev == null) return null;
            const next = selectedPeriode?.matieres.some(m => m.id === prev) ? prev : null;
            writeNum(kMatiere, next);
            return next;
        });
    }, [selectedPeriode, tree, kMatiere]);

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
        selectedPeriodeId,
        setSelectedPeriodeId,
        selectedPeriode,
        selectedMatiereId,
        setSelectedMatiereId,
    };
}
