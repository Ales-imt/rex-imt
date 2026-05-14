import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';

type Props = {
  onPrev?: () => void;
  onNext?: () => void;
  onSubmit?: () => void;
  nextDisabled?: boolean;
  nextLabel?: string;
};

export function NavButtons({ onPrev, onNext, onSubmit, nextDisabled = false, nextLabel }: Props) {
  return (
    <View style={styles.row}>
      {onPrev ? (
        <TouchableOpacity style={styles.prev} onPress={onPrev} activeOpacity={0.7}>
          <Text style={styles.prevText}>← Précédent</Text>
        </TouchableOpacity>
      ) : (
        <View style={styles.spacer} />
      )}

      {onNext && (
        <TouchableOpacity
          style={[styles.next, nextDisabled && styles.disabled]}
          onPress={nextDisabled ? undefined : onNext}
          activeOpacity={nextDisabled ? 1 : 0.7}
        >
          <Text style={styles.nextText}>{nextLabel ?? 'Suivant →'}</Text>
        </TouchableOpacity>
      )}

      {onSubmit && (
        <TouchableOpacity
          style={[styles.submit, nextDisabled && styles.disabled]}
          onPress={nextDisabled ? undefined : onSubmit}
          activeOpacity={nextDisabled ? 1 : 0.7}
        >
          <Text style={styles.nextText}>Soumettre</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', gap: 12, paddingHorizontal: 16, paddingVertical: 12 },
  spacer: { flex: 1 },
  prev: {
    flex: 1,
    height: 48,
    borderRadius: 12,
    borderWidth: 1.5,
    borderColor: '#CBD5E1',
    alignItems: 'center',
    justifyContent: 'center',
  },
  prevText: { fontSize: 15, fontWeight: '600', color: '#64748B' },
  next: {
    flex: 1,
    height: 48,
    borderRadius: 12,
    backgroundColor: '#6366F1',
    alignItems: 'center',
    justifyContent: 'center',
  },
  submit: {
    flex: 1,
    height: 48,
    borderRadius: 12,
    backgroundColor: '#22C55E',
    alignItems: 'center',
    justifyContent: 'center',
  },
  disabled: { opacity: 0.4 },
  nextText: { fontSize: 15, fontWeight: '600', color: '#fff' },
});
