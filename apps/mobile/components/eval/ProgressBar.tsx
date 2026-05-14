import { StyleSheet, View } from 'react-native';

type Props = { currentStep: number; totalSteps?: number };

export function ProgressBar({ currentStep, totalSteps = 7 }: Props) {
  return (
    <View style={styles.row}>
      {Array.from({ length: totalSteps }, (_, i) => (
        <View
          key={i}
          style={[styles.bar, { backgroundColor: i < currentStep ? '#6366F1' : 'rgba(255,255,255,0.3)' }]}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', gap: 4, paddingHorizontal: 16, paddingVertical: 10 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
});
