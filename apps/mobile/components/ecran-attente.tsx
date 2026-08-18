import { LogoRex } from '@/components/logo-rex';
import { PRIMARY } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { useEffect, useState } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';

// Au-delà de ce délai, la vérification n'est plus instantanée : on l'annonce.
// En deçà, afficher un indicateur reviendrait à faire clignoter l'écran pour
// une lecture de stockage de quelques millisecondes.
const DELAI_INDICATEUR_MS = 400;

/**
 * Écran montré tant que la validité de la session n'est pas tranchée
 * (cf. services/session.ts). Il ne peint ni contenu d'élève ni formulaire de
 * connexion : c'est précisément ce qui évite d'exposer la page du compte
 * précédent à quelqu'un dont le jeton est périmé, et le login à quelqu'un de
 * parfaitement connecté.
 */
export function EcranAttente() {
  const colors = useTheme();
  const [lent, setLent] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setLent(true), DELAI_INDICATEUR_MS);
    return () => clearTimeout(t);
  }, []);

  return (
    <View style={[styles.page, { backgroundColor: colors.pageBg }]}>
      <LogoRex size={72} />
      <View style={styles.attente}>
        {lent ? (
          <>
            <ActivityIndicator size="small" color={PRIMARY} />
            <Text style={[styles.texte, { color: colors.textSecondary }]}>Connexion…</Text>
          </>
        ) : null}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  page: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  // Hauteur réservée : l'indicateur apparaît sans déplacer le logo.
  attente: {
    height: 48,
    marginTop: 24,
    alignItems: 'center',
    gap: 8,
  },
  texte: {
    fontSize: 13,
  },
});
