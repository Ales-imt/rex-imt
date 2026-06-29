import { useTheme } from '@/hooks/use-theme';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { useEffect, useRef } from 'react';
import { StyleSheet, Text, View } from 'react-native';

type Props = {
  active: boolean;
  onScan: (code: string) => void;
};

export function PresenceScanner({ active, onScan }: Props) {
  const colors = useTheme();
  const [permission, requestPermission] = useCameraPermissions();
  const lastScanRef = useRef(0);

  useEffect(() => {
    if (permission && !permission.granted && permission.canAskAgain) {
      requestPermission();
    }
  }, [permission]);

  if (!permission) {
    return (
      <View style={[styles.box, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
        <Text style={[styles.message, { color: colors.textSecondary }]}>
          Vérification de la permission caméra…
        </Text>
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={[styles.box, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
        <Text style={[styles.message, { color: colors.textSecondary }]}>
          Caméra indisponible (permission refusée). Utilisez la saisie manuelle ci-dessous.
        </Text>
      </View>
    );
  }

  return (
    <View style={[styles.box, { borderColor: colors.cardBorder }]}>
      <CameraView
        style={styles.camera}
        facing="back"
        barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
        onBarcodeScanned={active ? ({ data }) => {
          const now = Date.now();
          if (now - lastScanRef.current < 1500) return;
          lastScanRef.current = now;
          onScan(data);
        } : undefined}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  box: {
    borderRadius: 12,
    borderWidth: 1,
    overflow: 'hidden',
    height: 260,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 16,
  },
  camera: {
    width: '100%',
    height: '100%',
  },
  message: {
    fontSize: 13,
    textAlign: 'center',
  },
});
