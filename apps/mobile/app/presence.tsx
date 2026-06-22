import { HeaderMenu } from '@/components/header-menu';
import { PresenceScanner } from '@/components/presence-scanner';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance, setupAxiosInterceptors } from '@/services/api';
import * as Haptics from 'expo-haptics';
import { Stack, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import {
    ActivityIndicator,
    StyleSheet,
    Text,
    TextInput,
    TouchableOpacity,
    View,
} from 'react-native';

type Etat = 'idle' | 'loading' | 'succes' | 'erreur';

type ReponsePresence = {
    matiere: string;
    seance_id: number;
    statut: string;
    pointe_at: string;
    deja_pointe: boolean;
};

export const PresenceScreen = () => {
    const colors = useTheme();
    const navMenu = useNavMenu('presence');
    const [etat, setEtat] = useState<Etat>('idle');
    const [codeManuel, setCodeManuel] = useState('');
    const [resultat, setResultat] = useState<ReponsePresence | null>(null);
    const [erreur, setErreur] = useState('');
    const [focused, setFocused] = useState(false);

    useFocusEffect(
        useCallback(() => {
            setupAxiosInterceptors();
            setFocused(true);
            return () => {
                setFocused(false);
            };
        }, [])
    );

    const envoyerCode = (code: string) => {
        const trimmed = code.trim();
        const codeNettoye = trimmed.includes('.') ? trimmed : trimmed.toUpperCase();
        if (!codeNettoye || etat === 'loading') return;
        setEtat('loading');
        setErreur('');
        apiInstance
            .post<ReponsePresence>('/pointage', { code: codeNettoye })
            .then(res => {
                setResultat(res.data ?? null);
                setEtat('succes');
                Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success).catch(() => { });
            })
            .catch(e => {
                const status = e?.response?.status;
                if (status === 404) {
                    setErreur('Séance introuvable. Vérifiez le code.');
                } else if (status === 409) {
                    setErreur('Séance fermée.');
                } else {
                    setErreur(e?.response?.data?.message ?? 'Code invalide ou séance fermée.');
                }
                setEtat('erreur');
            });
    };

    const reessayer = () => {
        setEtat('idle');
        setErreur('');
        setResultat(null);
        setCodeManuel('');
    };

    return (
        <>
            <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />
            <View style={[styles.container, { backgroundColor: colors.pageBg }]}>
                <ZenBackground />

                {etat === 'succes' ? (
                    <View style={styles.centerBox}>
                        <View style={[styles.successCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                            <Text style={styles.successIcon}>✅</Text>
                            <Text style={[styles.successTitle, { color: colors.textPrimary }]}>
                                Présence enregistrée
                            </Text>
                            {resultat?.matiere && (
                                <Text style={[styles.successDetail, { color: colors.textSecondary }]}>
                                    {resultat.matiere}
                                </Text>
                            )}
                            {resultat?.statut === 'RETARD' && (
                                <Text style={styles.retardBadge}>Retard</Text>
                            )}
                            {resultat?.deja_pointe && (
                                <Text style={[styles.dejaPointeText, { color: colors.textSecondary }]}>
                                    Déjà pointé
                                </Text>
                            )}
                            <TouchableOpacity
                                style={[styles.doneBtn, { backgroundColor: colors.tint }]}
                                onPress={reessayer}
                            >
                                <Text style={[styles.doneBtnText, { color: colors.background }]}>Terminé</Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                ) : (
                    <View style={styles.scroll}>
                        <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>
                            Scanner le QR code affiché en cours
                        </Text>

                        <PresenceScanner key={focused ? 'focused' : 'unfocused'} active={etat === 'idle' && focused} onScan={envoyerCode} />

                        {etat === 'erreur' && (
                            <View style={[styles.errorBox, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                                <Text style={styles.errorText}>{erreur}</Text>
                                <TouchableOpacity style={[styles.retryBtn, { backgroundColor: colors.tint }]} onPress={reessayer}>
                                    <Text style={[styles.retryBtnText, { color: colors.background }]}>Réessayer</Text>
                                </TouchableOpacity>
                            </View>
                        )}

                        <Text style={[styles.sectionTitle, { color: colors.textPrimary, marginTop: 24 }]}>
                            Ou saisir le code manuellement
                        </Text>
                        <View style={[styles.manualCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                            <TextInput
                                style={[styles.manualInput, { color: colors.textPrimary, borderColor: colors.inputBorderLogin }]}
                                value={codeManuel}
                                onChangeText={setCodeManuel}
                                placeholder="ex. 7K2-9F"
                                placeholderTextColor={colors.textSecondary}
                                autoCapitalize="characters"
                                autoCorrect={false}
                                editable={etat !== 'loading'}
                                onSubmitEditing={() => envoyerCode(codeManuel)}
                            />
                            <TouchableOpacity
                                style={[styles.validateBtn, { backgroundColor: colors.tint }, etat === 'loading' && styles.btnDisabled]}
                                onPress={() => envoyerCode(codeManuel)}
                                disabled={etat === 'loading'}
                            >
                                {etat === 'loading' ? (
                                    <ActivityIndicator color={colors.background} />
                                ) : (
                                    <Text style={[styles.validateBtnText, { color: colors.background }]}>Valider</Text>
                                )}
                            </TouchableOpacity>
                        </View>
                    </View>
                )}
            </View>
        </>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
    },
    scroll: {
        padding: 16,
        gap: 8,
    },
    sectionTitle: {
        fontSize: 15,
        fontWeight: '600',
        marginBottom: 8,
    },
    manualCard: {
        flexDirection: 'row',
        alignItems: 'center',
        borderRadius: 10,
        borderWidth: 1,
        padding: 12,
        gap: 10,
    },
    manualInput: {
        flex: 1,
        borderWidth: 1,
        borderRadius: 8,
        paddingHorizontal: 12,
        paddingVertical: 10,
        fontSize: 15,
        letterSpacing: 1,
    },
    validateBtn: {
        paddingHorizontal: 18,
        paddingVertical: 12,
        borderRadius: 8,
        minWidth: 84,
        alignItems: 'center',
    },
    btnDisabled: {
        opacity: 0.6,
    },
    validateBtnText: {
        fontSize: 14,
        fontWeight: '600',
    },
    errorBox: {
        borderRadius: 10,
        borderWidth: 1,
        padding: 14,
        marginTop: 12,
        gap: 10,
        alignItems: 'center',
    },
    errorText: {
        color: '#d32f2f',
        fontSize: 14,
        textAlign: 'center',
    },
    retryBtn: {
        paddingHorizontal: 20,
        paddingVertical: 10,
        borderRadius: 8,
    },
    retryBtnText: {
        fontSize: 14,
        fontWeight: '600',
    },
    centerBox: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        padding: 24,
    },
    successCard: {
        borderRadius: 14,
        borderWidth: 1,
        padding: 28,
        alignItems: 'center',
        gap: 8,
        minWidth: 260,
    },
    successIcon: {
        fontSize: 40,
        marginBottom: 4,
    },
    successTitle: {
        fontSize: 17,
        fontWeight: '700',
    },
    successDetail: {
        fontSize: 14,
        textAlign: 'center',
    },
    retardBadge: {
        fontSize: 13,
        fontWeight: '600',
        color: '#e65100',
        backgroundColor: '#fff3e0',
        paddingHorizontal: 10,
        paddingVertical: 3,
        borderRadius: 6,
    },
    dejaPointeText: {
        fontSize: 13,
        fontStyle: 'italic',
    },
    doneBtn: {
        marginTop: 16,
        paddingHorizontal: 28,
        paddingVertical: 12,
        borderRadius: 8,
    },
    doneBtnText: {
        fontSize: 15,
        fontWeight: '600',
    },
});

export default PresenceScreen;
