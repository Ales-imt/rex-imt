import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';

type Props = { value: number | null; onChange: (v: number) => void; size?: number };

export function StarRating({ value, onChange, size = 38 }: Props) {
  return (
    <View style={styles.row}>
      {[1, 2, 3, 4, 5].map((n) => (
        <TouchableOpacity key={n} onPress={() => onChange(n)} activeOpacity={0.7}>
          <Text style={{ fontSize: size, color: (value ?? 0) >= n ? '#F59E0B' : '#CBD5E1' }}>★</Text>
        </TouchableOpacity>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', gap: 6 },
});
