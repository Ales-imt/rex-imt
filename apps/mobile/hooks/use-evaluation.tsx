import React, { createContext, useContext, useReducer } from 'react';

export type EvaluationState = {
  matiereId: string;
  formatSuivi: 'PRESENTIEL' | 'DISTANCIEL' | 'HYBRIDE' | null;
  tendanceSemestre: 'PROGRES' | 'STABLE' | 'BAISSE' | null;
  assiduite: 'TOUTES' | 'QUELQUES_ABSENCES' | 'BEAUCOUP' | null;
  scorePedaGlobal: number | null;
  scorePedaClarte: number | null;
  scoreContenuAdequation: number | null;
  chipsContenu: string[];
  chargePerception: 'LEGERE' | 'ADAPTEE' | 'LOURDE' | null;
  chargeHeuresSemaine: 'H_MOINS_1' | 'H_1_3' | 'H_3_5' | 'H_PLUS_5' | null;
  scoreDifficulte: number | null;
  scoreSupports: number | null;
  chipsSupports: string[];
  verbatimSupports: string;
  scoreAmbiance: number | null;
  chipsAmbiance: string[];
  verbatimAmbiance: string;
  nps: number | null;
  verbatimNps: string;
};

const INITIAL: EvaluationState = {
  matiereId: '',
  formatSuivi: null,
  tendanceSemestre: null,
  assiduite: null,
  scorePedaGlobal: null,
  scorePedaClarte: null,
  scoreContenuAdequation: null,
  chipsContenu: [],
  chargePerception: null,
  chargeHeuresSemaine: null,
  scoreDifficulte: null,
  scoreSupports: null,
  chipsSupports: [],
  verbatimSupports: '',
  scoreAmbiance: null,
  chipsAmbiance: [],
  verbatimAmbiance: '',
  nps: null,
  verbatimNps: '',
};

type Action =
  | { type: 'SET'; payload: Partial<EvaluationState> }
  | { type: 'INIT'; matiereId: string }
  | { type: 'RESET' };

function reducer(state: EvaluationState, action: Action): EvaluationState {
  switch (action.type) {
    case 'SET': return { ...state, ...action.payload };
    case 'INIT': return { ...INITIAL, matiereId: action.matiereId };
    case 'RESET': return INITIAL;
  }
}

type ContextValue = {
  state: EvaluationState;
  set: (partial: Partial<EvaluationState>) => void;
  init: (matiereId: string) => void;
  reset: () => void;
};

export const EvaluationContext = createContext<ContextValue | null>(null);

export function EvaluationProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(reducer, INITIAL);
  const value: ContextValue = {
    state,
    set: (payload) => dispatch({ type: 'SET', payload }),
    init: (matiereId) => dispatch({ type: 'INIT', matiereId }),
    reset: () => dispatch({ type: 'RESET' }),
  };
  return <EvaluationContext.Provider value={value}>{children}</EvaluationContext.Provider>;
}

export function useEvaluation(): ContextValue {
  const ctx = useContext(EvaluationContext);
  if (!ctx) throw new Error('useEvaluation doit être utilisé dans EvaluationProvider');
  return ctx;
}
