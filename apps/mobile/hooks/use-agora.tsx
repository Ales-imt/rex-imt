import React, { createContext, useContext, useState } from 'react';

type AgoraContextValue = {
    months: number;
    promotion: string;
    setMonths: (months: number) => void;
    setPromotion: (promotion: string) => void;
};

const AgoraContext = createContext<AgoraContextValue | null>(null);

export function AgoraProvider({ children }: { children: React.ReactNode }) {
    const [months, setMonths] = useState(1);
    const [promotion, setPromotion] = useState('');

    return (
        <AgoraContext.Provider value={{ months, promotion, setMonths, setPromotion }}>
            {children}
        </AgoraContext.Provider>
    );
}

export function useAgora(): AgoraContextValue {
    const ctx = useContext(AgoraContext);
    if (!ctx) throw new Error('useAgora doit être utilisé dans AgoraProvider');
    return ctx;
}
