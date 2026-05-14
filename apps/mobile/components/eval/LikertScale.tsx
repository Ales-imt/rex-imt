import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';

type Props = {
  value: number | null;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  labelMin?: string;
  labelMax?: string;
};

export function LikertScale({ value, onChange, min = 1, max = 5, labelMin, labelMax }: Props) {
  const items = Array.from({ length: max - min + 1 }, (_, i) => i + min);
  return (
    <View style={styles.container}>
      <View style={styles.row}>
        {items.map((n) => {
          const active = value === n;
          return (
            <TouchableOpacity
              key={n}
              style={[styles.dot, active && styles.dotActive]}
              onPress={() => onChange(n)}
              activeOpacity={0.7}
            >
              <Text style={[styles.label, active && styles.labelActive]}>{n}</Text>
            </TouchableOpacity>
          );
        })}
      </View>
      {(labelMin || labelMax) && (
        <View style={styles.legends}>
          <Text style={styles.legend}>{labelMin}</Text>
          <Text style={styles.legend}>{labelMax}</Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: 8 },
  row: { flexDirection: 'row', gap: 8, justifyContent: 'center' },
  dot: {
    width: 44,
    height: 44,
    borderRadius: 22,
    borderWidth: 1.5,
    borderColor: '#CBD5E1',
    alignItems: 'center',
    justifyContent: 'center',
  },
  dotActive: { backgroundColor: '#6366F1', borderColor: '#6366F1' },
  label: { fontSize: 15, fontWeight: '600', color: '#64748B' },
  labelActive: { color: '#fff' },
  legends: { flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 4 },
  legend: { fontSize: 11, color: '#94A3B8' },
});
