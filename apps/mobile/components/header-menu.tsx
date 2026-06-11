import { useState } from 'react';
import {
  Modal,
  Pressable,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';

export type MenuItem = {
  label: string;
  icon: string;
  onPress: () => void;
  danger?: boolean;
};

export function HeaderMenu({ items }: { items: MenuItem[] }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <TouchableOpacity style={styles.trigger} onPress={() => setOpen(true)} hitSlop={8}>
        <Text style={styles.dots}>⋮</Text>
      </TouchableOpacity>

      <Modal visible={open} transparent animationType="fade" onRequestClose={() => setOpen(false)}>
        <Pressable style={styles.overlay} onPress={() => setOpen(false)}>
          <View style={styles.menu}>
            {items.map((item, i) => (
              <TouchableOpacity
                key={i}
                style={[styles.item, i < items.length - 1 && styles.itemBorder]}
                onPress={() => { setOpen(false); item.onPress(); }}
                activeOpacity={0.7}
              >
                <Text style={styles.itemIcon}>{item.icon}</Text>
                <Text style={[styles.itemLabel, item.danger && styles.itemDanger]}>
                  {item.label}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </Pressable>
      </Modal>
    </>
  );
}

const styles = StyleSheet.create({
  trigger: {
    paddingHorizontal: 12,
    paddingVertical: 4,
  },
  dots: {
    fontSize: 22,
    color: '#1976d2',
    fontWeight: '700',
    lineHeight: 24,
  },
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.15)',
    alignItems: 'flex-end',
    paddingTop: 52,
    paddingRight: 8,
  },
  menu: {
    backgroundColor: '#fff',
    borderRadius: 8,
    minWidth: 180,
    boxShadow: '0 4px 8px rgba(0,0,0,0.15)',
    elevation: 8,
    overflow: 'hidden',
  },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 12,
  },
  itemBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(0,0,0,0.1)',
  },
  itemIcon: {
    fontSize: 18,
    width: 24,
    textAlign: 'center',
  },
  itemLabel: {
    fontSize: 15,
    color: '#222',
  },
  itemDanger: {
    color: '#d32f2f',
  },
});
