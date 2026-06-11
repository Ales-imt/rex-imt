import { Colors } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { API_BASE, login } from '@/services/auth';
import { apiInstance } from '@/services/api';
import { getPseudo, saveTokens } from '@/services/tokens';
import { useRouter } from 'expo-router';
import {  useMemo, useState } from 'react';
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
  keyboardType?: 'email-address' | 'default';
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
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const dynamicStyles = useMemo(() => StyleSheet.create({
    page: { backgroundColor: colors.pageBg },
    card: { backgroundColor: colors.cardBg, borderColor: colors.cardBorder },
    dividerLine: { backgroundColor: colors.dividerLine },
  }), [colors]);


  async function handleSignIn() {
    if (!email || !password) {
      setError('Veuillez renseigner tous les champs.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const resp = await login(email, password);
      await saveTokens(resp.access_token, resp.refresh_token);
      const pseudo = await getPseudo();
      if (!pseudo) {
        router.replace('/pseudo-setup');
        return;
      }
      const me = await apiInstance.get('/me');
      if (!me.data.informed_at) {
        router.replace('/apropos-first-login');
      } else {
        router.replace('/agora');
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Erreur de connexion');
    } finally {
      setLoading(false);
    }
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
            <Text style={[styles.subtitle, { color: colors.textSecondary }]}>pour continuer sur Mon App</Text>

            {error ? (
              <View style={styles.alert}>
                <Text style={styles.alertText}>{error}</Text>
              </View>
            ) : null}

            <View style={styles.form}>
              <LabeledInput
                label="Adresse e-mail"
                value={email}
                onChangeText={setEmail}
                keyboardType="email-address"
                colors={colors}
              />
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
            </View>
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
});
