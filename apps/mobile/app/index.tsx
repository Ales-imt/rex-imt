import { Colors } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { login, requestEmailCode, verifyEmailCode, type LoginResponse } from '@/services/auth';
import { apiInstance } from '@/services/api';
import { getPseudo, saveRoles, saveTokens } from '@/services/tokens';
import { useRouter } from 'expo-router';
import { useMemo, useState } from 'react';
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

const PRIMARY = '#1976d2';

// Domaine interne (LDAP) : mines-ales.fr et ses sous-domaines (etu.mines-ales.fr, ...).
// Purement indicatif côté front — le backend ne fait jamais confiance à ce
// routage : /auth/login reste strictement LDAP, /auth/email/* ne sert que les
// comptes auth_source='email'.
function getEmailDomain(email: string): string {
  const at = email.lastIndexOf('@');
  if (at === -1) return '';
  return email.slice(at + 1).toLowerCase();
}

function isInternalDomain(email: string): boolean {
  const domain = getEmailDomain(email);
  return domain === 'mines-ales.fr' || domain.endsWith('.mines-ales.fr');
}

type Step = 'identify' | 'password' | 'external' | 'code';

function LabeledInput({
  label,
  value,
  onChangeText,
  secureTextEntry = false,
  keyboardType,
  autoCapitalize = 'none',
  colors,
}: {
  label: string;
  value: string;
  onChangeText: (v: string) => void;
  secureTextEntry?: boolean;
  keyboardType?: 'email-address' | 'default' | 'number-pad';
  autoCapitalize?: 'none' | 'sentences';
  colors: typeof Colors.light;
}) {
  const [focused, setFocused] = useState(false);

  return (
    <View style={styles.fieldGroup}>
      <Text style={[styles.label, { color: colors.textPrimary }]}>{label} *</Text>
      <TextInput
        style={[
          styles.input,
          { borderColor: colors.inputBorderLogin, color: colors.textPrimary },
          focused && { borderColor: PRIMARY, borderWidth: 2 },
        ]}
        value={value}
        onChangeText={onChangeText}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        secureTextEntry={secureTextEntry}
        keyboardType={keyboardType}
        autoCapitalize={autoCapitalize}
        placeholderTextColor={colors.textSecondary}
      />
    </View>
  );
}

export default function SignInScreen() {
  const router = useRouter();
  const colors = useTheme();
  const [step, setStep] = useState<Step>('identify');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const dynamicStyles = useMemo(() => StyleSheet.create({
    page: { backgroundColor: colors.pageBg },
    card: { backgroundColor: colors.cardBg, borderColor: colors.cardBorder },
    dividerLine: { backgroundColor: colors.dividerLine },
  }), [colors]);

  async function afterLoginSuccess(resp: LoginResponse) {
    await saveTokens(resp.access_token, resp.refresh_token);
    await saveRoles(resp.roles ?? []);
    const pseudo = await getPseudo();
    if (!pseudo) {
      router.replace('/pseudo-setup');
      return;
    }
    const me = await apiInstance.get('/me');
    if (!me.data.informed_at) {
      router.replace('/apropos-first-login');
    } else {
      router.replace('/programme');
    }
  }

  function handleContinue() {
    if (!email) {
      setError('Veuillez renseigner votre adresse e-mail.');
      return;
    }
    setError('');
    setStep(isInternalDomain(email) ? 'password' : 'external');
  }

  async function handleSignIn() {
    if (!password) {
      setError('Veuillez renseigner votre mot de passe.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const resp = await login(email, password);
      await afterLoginSuccess(resp);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Erreur de connexion');
    } finally {
      setLoading(false);
    }
  }

  async function handleRequestCode() {
    setError('');
    setLoading(true);
    try {
      await requestEmailCode(email);
      setCode('');
      setStep('code');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Erreur lors de l’envoi du code');
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyCode() {
    if (!code) {
      setError('Veuillez renseigner le code reçu par e-mail.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const resp = await verifyEmailCode(email, code);
      await afterLoginSuccess(resp);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Code invalide');
    } finally {
      setLoading(false);
    }
  }

  function handleChangeEmail() {
    setError('');
    setPassword('');
    setCode('');
    setStep('identify');
  }

  return (
    <SafeAreaView style={[styles.page, dynamicStyles.page]} edges={['top', 'bottom']}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <View style={[styles.card, dynamicStyles.card]}>
            <View style={styles.logoContainer}>
              <View style={styles.logo}>
                <Text style={styles.logoText}>A</Text>
              </View>
            </View>

            <Text style={[styles.title, { color: colors.textPrimary }]}>Connexion</Text>
            <Text style={[styles.subtitle, { color: colors.textSecondary }]}>pour continuer sur REX</Text>

            {error ? (
              <View style={styles.alert}>
                <Text style={styles.alertText}>{error}</Text>
              </View>
            ) : null}

            {step === 'identify' && (
              <View style={styles.form}>
                <LabeledInput
                  label="Adresse e-mail"
                  value={email}
                  onChangeText={setEmail}
                  keyboardType="email-address"
                  colors={colors}
                />
                <TouchableOpacity style={[styles.signInBtn, loading && { opacity: 0.6 }]} onPress={handleContinue} activeOpacity={0.85} disabled={loading}>
                  <Text style={styles.signInBtnText}>Continuer</Text>
                </TouchableOpacity>
              </View>
            )}

            {step === 'password' && (
              <View style={styles.form}>
                <Text style={[styles.subtitle, { color: colors.textSecondary, marginBottom: 0 }]}>{email}</Text>
                <LabeledInput
                  label="Mot de passe"
                  value={password}
                  onChangeText={setPassword}
                  secureTextEntry
                  colors={colors}
                />
                <TouchableOpacity style={[styles.signInBtn, loading && { opacity: 0.6 }]} onPress={handleSignIn} activeOpacity={0.85} disabled={loading}>
                  <Text style={styles.signInBtnText}>{loading ? 'Connexion…' : 'Se connecter'}</Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={handleChangeEmail} disabled={loading}>
                  <Text style={[styles.linkText, { color: colors.textSecondary }]}>Changer d’adresse e-mail</Text>
                </TouchableOpacity>
              </View>
            )}

            {step === 'external' && (
              <View style={styles.form}>
                <Text style={[styles.subtitle, { color: colors.textSecondary, marginBottom: 0 }]}>{email}</Text>
                <Text style={[styles.helperText, { color: colors.textSecondary }]}>
                  Adresse externe détectée : un code de connexion à usage unique vous sera envoyé par e-mail.
                </Text>
                <TouchableOpacity style={[styles.signInBtn, loading && { opacity: 0.6 }]} onPress={handleRequestCode} activeOpacity={0.85} disabled={loading}>
                  <Text style={styles.signInBtnText}>{loading ? 'Envoi…' : 'Recevoir un code'}</Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={handleChangeEmail} disabled={loading}>
                  <Text style={[styles.linkText, { color: colors.textSecondary }]}>Changer d’adresse e-mail</Text>
                </TouchableOpacity>
              </View>
            )}

            {step === 'code' && (
              <View style={styles.form}>
                <Text style={[styles.subtitle, { color: colors.textSecondary, marginBottom: 0 }]}>{email}</Text>
                <LabeledInput
                  label="Code reçu par e-mail"
                  value={code}
                  onChangeText={setCode}
                  keyboardType="number-pad"
                  colors={colors}
                />
                <TouchableOpacity style={[styles.signInBtn, loading && { opacity: 0.6 }]} onPress={handleVerifyCode} activeOpacity={0.85} disabled={loading}>
                  <Text style={styles.signInBtnText}>{loading ? 'Vérification…' : 'Valider'}</Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={handleRequestCode} disabled={loading}>
                  <Text style={[styles.linkText, { color: colors.textSecondary }]}>Renvoyer un code</Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={handleChangeEmail} disabled={loading}>
                  <Text style={[styles.linkText, { color: colors.textSecondary }]}>Changer d’adresse e-mail</Text>
                </TouchableOpacity>
              </View>
            )}
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  page: {
    flex: 1,
  },
  flex: {
    flex: 1,
  },
  scroll: {
    flexGrow: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: 40,
    paddingHorizontal: 16,
  },
  card: {
    width: '100%',
    maxWidth: 380,
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 32,
    paddingVertical: 36,
    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    elevation: 4,
  },
  logoContainer: {
    alignItems: 'center',
    marginBottom: 16,
  },
  logo: {
    width: 40,
    height: 40,
    borderRadius: 8,
    backgroundColor: PRIMARY,
    justifyContent: 'center',
    alignItems: 'center',
  },
  logoText: {
    color: '#fff',
    fontSize: 20,
    fontWeight: '700',
  },
  title: {
    fontSize: 22,
    fontWeight: '600',
    textAlign: 'center',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 13,
    textAlign: 'center',
    marginBottom: 24,
  },
  helperText: {
    fontSize: 12,
    textAlign: 'center',
  },
  alert: {
    backgroundColor: '#fdecea',
    borderRadius: 4,
    padding: 12,
    marginBottom: 16,
    borderLeftWidth: 4,
    borderLeftColor: '#d32f2f',
  },
  alertText: {
    color: '#d32f2f',
    fontSize: 13,
  },
  form: {
    gap: 12,
  },
  fieldGroup: {
    gap: 4,
  },
  label: {
    fontSize: 13,
    fontWeight: '500',
  },
  input: {
    borderWidth: 1,
    borderRadius: 4,
    paddingHorizontal: 12,
    paddingVertical: Platform.OS === 'ios' ? 12 : 10,
    fontSize: 14,
  },
  signInBtn: {
    backgroundColor: PRIMARY,
    borderRadius: 4,
    paddingVertical: 12,
    alignItems: 'center',
    marginTop: 4,
    boxShadow: '0 2px 4px rgba(25,118,210,0.3)',
    elevation: 3,
  },
  signInBtnText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
    letterSpacing: 0.5,
  },
  linkText: {
    fontSize: 12,
    textAlign: 'center',
    textDecorationLine: 'underline',
  },
});
