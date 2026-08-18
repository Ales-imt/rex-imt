import { PRIMARY } from '@/constants/theme';
import { StyleSheet, Text, View } from 'react-native';

/**
 * Marque de l'application, partagée par l'écran d'attente et le formulaire de
 * connexion : les deux se succèdent au démarrage, et un logo qui change de
 * taille ou de couleur entre eux se lit comme un clignotement.
 */
export function LogoRex({ size = 40 }: { size?: number }) {
  return (
    <View style={[styles.logo, { width: size, height: size, borderRadius: size * 0.2 }]}>
      <Text style={[styles.texte, { fontSize: size * 0.5 }]}>A</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  logo: {
    backgroundColor: PRIMARY,
    justifyContent: 'center',
    alignItems: 'center',
  },
  texte: {
    color: '#fff',
    fontWeight: '700',
  },
});
