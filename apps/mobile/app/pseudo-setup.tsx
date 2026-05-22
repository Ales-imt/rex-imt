import { ZenBackground } from '@/components/zen-background';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { generatePseudo, savePseudo } from '@/services/tokens';
import * as Clipboard from 'expo-clipboard';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import {
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function PseudoSetupScreen() {
  const colors = useTheme();
  const router = useRouter();

  const [pseudo] = useState(() => generatePseudo());
  const [copied, setCopied] = useState(false);
  const [showExisting, setShowExisting] = useState(false);
  const [existingInput, setExistingInput] = useState('');
  const [inputError, setInputError] = useState('');

  async function handleCopy() {
    await Clipboard.setStringAsync(pseudo);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function navigateAfterPseudo() {
    const me = await apiInstance.get('/me');
    if (!me.data.informed_at) {
      router.replace('/apropos-first-login');
    } else {
      router.replace('/agora');
    }
  }

  async function handleContinueNew() {
    await savePseudo(pseudo);
    await navigateAfterPseudo();
  }

  async function handleContinueExisting() {
    const trimmed = existingInput.trim().toLowerCase();
    const parts = trimmed.split(/\s*\.\s*/);
    if (parts.length !== 4 || parts.some(p => p.length === 0)) {
      setInputError('Format attendu : mot . mot . mot . mot (4 mots séparés par .)');
      return;
    }
    await savePseudo(trimmed);
    await navigateAfterPseudo();
  }

  return (
    <SafeAreaView style={[styles.page, { backgroundColor: colors.pageBg }]} edges={['top', 'bottom']}>
      <ZenBackground />
      <ScrollView
        contentContainerStyle={styles.scroll}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        <View style={[styles.card, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>

          {/* ── En-tête ── */}
          <View style={styles.header}>
            <View style={[styles.icon, { backgroundColor: colors.tint }]}>
              <Text style={styles.iconText}>🔑</Text>
            </View>
            <Text style={[styles.title, { color: colors.textPrimary }]}>Votre pseudo anonyme</Text>
            <Text style={[styles.subtitle, { color: colors.textSecondary }]}>
              Ce pseudo identifie vos feedbacks sans révéler votre identité.
              Personne d'autre ne peut le connaître.
            </Text>
          </View>

          {!showExisting ? (
            <>
              {/* ── Chemin A : nouveau pseudo ── */}
              <View style={[styles.pseudoBox, { backgroundColor: colors.tint + '18', borderColor: colors.tint + '55' }]}>
                <Text style={[styles.pseudoText, { color: colors.tint }]}>{pseudo}</Text>
                <TouchableOpacity style={styles.copyBtn} onPress={handleCopy} activeOpacity={0.7}>
                  <Text style={[styles.copyBtnText, { color: colors.tint }]}>
                    {copied ? '✓ Copié !' : '📋 Copier'}
                  </Text>
                </TouchableOpacity>
              </View>

              <View style={[styles.warningBox, { backgroundColor: colors.cardBorder + '55', borderColor: colors.dividerLine }]}>
                <Text style={[styles.warningText, { color: colors.textSecondary }]}>
                  ⚠️  Notez ces mots précieusement. Ils vous permettront de retrouver vos feedbacks sur un autre appareil.{' '}
                  <Text style={{ fontWeight: '700', color: colors.textPrimary }}>Nous ne pouvons pas les récupérer à votre place.</Text>
                </Text>
              </View>

              <TouchableOpacity
                style={[styles.primaryBtn, { backgroundColor: colors.tint }]}
                onPress={handleContinueNew}
                activeOpacity={0.85}
              >
                <Text style={[styles.primaryBtnText, { color: colors.background }]}>J'ai noté mon pseudo, continuer</Text>
              </TouchableOpacity>

              <TouchableOpacity style={styles.secondaryLink} onPress={() => setShowExisting(true)} activeOpacity={0.7}>
                <Text style={[styles.secondaryLinkText, { color: colors.textSecondary }]}>
                  J'ai déjà un pseudo →
                </Text>
              </TouchableOpacity>
            </>
          ) : (
            <>
              {/* ── Chemin B : pseudo existant ── */}
              <Text style={[styles.fieldLabel, { color: colors.textPrimary }]}>
                Entrez votre pseudo (4 mots séparés par .)
              </Text>
              <TextInput
                style={[
                  styles.input,
                  { borderColor: colors.inputBorderLogin, color: colors.textPrimary },
                  inputError ? { borderColor: '#d32f2f' } : null,
                ]}
                value={existingInput}
                onChangeText={v => { setExistingInput(v); setInputError(''); }}
                placeholder="ex : cheval . nuage . fenêtre . bleu"
                placeholderTextColor={colors.textSecondary}
                autoCapitalize="none"
                autoCorrect={false}
              />
              {inputError ? (
                <Text style={styles.errorText}>{inputError}</Text>
              ) : null}

              <TouchableOpacity
                style={[styles.primaryBtn, { backgroundColor: colors.tint }]}
                onPress={handleContinueExisting}
                activeOpacity={0.85}
              >
                <Text style={[styles.primaryBtnText, { color: colors.background }]}>Utiliser ce pseudo</Text>
              </TouchableOpacity>

              <TouchableOpacity style={styles.secondaryLink} onPress={() => { setShowExisting(false); setInputError(''); }} activeOpacity={0.7}>
                <Text style={[styles.secondaryLinkText, { color: colors.textSecondary }]}>
                  ← Générer un nouveau pseudo
                </Text>
              </TouchableOpacity>
            </>
          )}

        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  page: {
    flex: 1,
  },
  scroll: {
    flexGrow: 1,
    justifyContent: 'center',
    paddingVertical: 40,
    paddingHorizontal: 16,
  },
  card: {
    borderRadius: 12,
    borderWidth: 1,
    paddingHorizontal: 24,
    paddingVertical: 32,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.08,
    shadowRadius: 8,
    elevation: 3,
  },

  // En-tête
  header: {
    alignItems: 'center',
    marginBottom: 28,
  },
  icon: {
    width: 52,
    height: 52,
    borderRadius: 14,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 14,
  },
  iconText: {
    fontSize: 26,
  },
  title: {
    fontSize: 20,
    fontWeight: '700',
    textAlign: 'center',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 13,
    textAlign: 'center',
    lineHeight: 20,
  },

  // Encadré pseudo
  pseudoBox: {
    borderWidth: 1.5,
    borderRadius: 10,
    paddingVertical: 20,
    paddingHorizontal: 16,
    alignItems: 'center',
    marginBottom: 16,
    gap: 12,
  },
  pseudoText: {
    fontSize: 22,
    fontWeight: '700',
    textAlign: 'center',
    letterSpacing: 0.5,
  },
  copyBtn: {
    paddingVertical: 6,
    paddingHorizontal: 14,
  },
  copyBtnText: {
    fontSize: 14,
    fontWeight: '600',
  },

  // Avertissement
  warningBox: {
    borderWidth: 1,
    borderRadius: 8,
    padding: 12,
    marginBottom: 20,
  },
  warningText: {
    fontSize: 13,
    lineHeight: 20,
  },

  // Bouton principal
  primaryBtn: {
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 16,
  },
  primaryBtnText: {
    fontSize: 15,
    fontWeight: '600',
    letterSpacing: 0.3,
  },

  // Lien secondaire
  secondaryLink: {
    alignItems: 'center',
    paddingVertical: 8,
  },
  secondaryLinkText: {
    fontSize: 13,
  },

  // Saisie chemin B
  fieldLabel: {
    fontSize: 13,
    fontWeight: '500',
    marginBottom: 8,
  },
  input: {
    borderWidth: 1,
    borderRadius: 6,
    paddingHorizontal: 12,
    paddingVertical: Platform.OS === 'ios' ? 12 : 10,
    fontSize: 15,
    marginBottom: 8,
    letterSpacing: 0.3,
  },
  errorText: {
    color: '#d32f2f',
    fontSize: 12,
    marginBottom: 12,
  },
});
