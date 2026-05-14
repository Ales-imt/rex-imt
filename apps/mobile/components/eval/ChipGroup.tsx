import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';

type Polarity = 'positive' | 'negative' | 'neutral';

type Props = {
  chips: { id: string; label: string }[];
  selected: string[];
  onToggle: (id: string) => void;
  polarity?: Polarity;
  multiSelect?: boolean;
};

const PALETTE: Record<Polarity, { bg: string; border: string; activeBg: string }> = {
  positive: { bg: '#ECFDF5', border: '#10B981', activeBg: '#10B981' },
  negative: { bg: '#FFF1F2', border: '#F43F5E', activeBg: '#F43F5E' },
  neutral:  { bg: '#EEF2FF', border: '#6366F1', activeBg: '#6366F1' },
};

export function ChipGroup({ chips, selected, onToggle, polarity = 'neutral', multiSelect = true }: Props) {
  const c = PALETTE[polarity];

  function handlePress(id: string) {
    if (!multiSelect) {
      onToggle(id);
      return;
    }
    onToggle(id);
  }

  return (
    <View style={styles.wrap}>
      {chips.map(({ id, label }) => {
        const active = selected.includes(id);
        return (
          <TouchableOpacity
            key={id}
            onPress={() => handlePress(id)}
            activeOpacity={0.7}
            style={[
              styles.chip,
              { backgroundColor: active ? c.activeBg : c.bg, borderColor: active ? c.activeBg : c.border },
            ]}
          >
            <Text style={[styles.text, { color: active ? '#fff' : '#334155' }]}>{label}</Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { borderWidth: 1.5, borderRadius: 20, paddingHorizontal: 14, paddingVertical: 8 },
  text: { fontSize: 13, fontWeight: '500' },
});
