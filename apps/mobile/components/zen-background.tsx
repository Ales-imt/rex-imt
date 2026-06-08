import { useTheme } from '@/hooks/use-theme';
import { StyleSheet, View } from 'react-native';

export function ZenBackground() {
  const colors = useTheme();
  return (
    <View style={[StyleSheet.absoluteFill, styles.container]} pointerEvents="none">
      <View style={[styles.base, { backgroundColor: colors.zenBase }]} />
      <View style={[styles.blob, styles.blob1, { backgroundColor: colors.zenBlob1 }]} />
      <View style={[styles.blob, styles.blob2, { backgroundColor: colors.zenBlob2 }]} />
      <View style={[styles.blob, styles.blob3, { backgroundColor: colors.zenBlob3 }]} />
      <View style={[styles.blob, styles.blob4, { backgroundColor: colors.zenBlob4 }]} />
      <View style={[styles.blob, styles.blob5, { backgroundColor: colors.zenBlob5 }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    overflow: 'hidden',
  },
  base: {
    ...StyleSheet.absoluteFillObject,
  },
  blob: {
    position: 'absolute',
    borderRadius: 999,
    opacity: 0.35,
  },
  blob1: { width: 320, height: 320, top: -80, left: -100 },
  blob2: { width: 260, height: 260, top: 60, right: -80 },
  blob3: { width: 200, height: 200, top: '38%', left: '20%', opacity: 0.22 },
  blob4: { width: 280, height: 280, bottom: 60, left: -60 },
  blob5: { width: 340, height: 340, bottom: -120, right: -80, opacity: 0.28 },
});
