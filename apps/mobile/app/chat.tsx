import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { Colors } from '@/constants/theme';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useOrientation } from '@/hooks/use-orientation';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { generateUUID, getPseudo } from '@/services/tokens';
import { Stack, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Keyboard,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  useWindowDimensions,
  View,
} from 'react-native';
import { FlashList } from "@shopify/flash-list";
import { SafeAreaView, type EdgeRecord } from 'react-native-safe-area-context';

type Message = {
  id: string;
  text: string;
  from: 'me' | 'other';
  time: string;
};

type ChatItem = {
  id: string;
  text: string;
  source: 'me' | 'other';
  ts: string;
};

const MONTH_OPTIONS = [1, 3, 6, 12];

function DropdownModal({
  visible, onClose, options, selected, onSelect,
}: {
  visible: boolean;
  onClose: () => void;
  options: { value: number; label: string }[];
  selected: number;
  onSelect: (v: number) => void;
}) {
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={styles.overlay} onPress={onClose}>
        <Pressable style={styles.dropdown}>
          <Text style={styles.dropdownTitle}>Période</Text>
          {options.map(opt => (
            <TouchableOpacity
              key={opt.value}
              style={[styles.dropdownItem, opt.value === selected && styles.dropdownItemActive]}
              onPress={() => { onSelect(opt.value); onClose(); }}
            >
              <Text style={[styles.dropdownItemText, opt.value === selected && styles.dropdownItemTextActive]}>
                {opt.label}
              </Text>
            </TouchableOpacity>
          ))}
        </Pressable>
      </Pressable>
    </Modal>
  );
}

function now() {
  return new Date().toISOString();
}

function formatTime(time: string): string {
  const d = new Date(time);
  if (isNaN(d.getTime())) return time;
  return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
}

function isToday(time: string): boolean {
  const d = new Date(time);
  if (isNaN(d.getTime())) return true;
  const t = new Date();
  return d.getDate() === t.getDate() && d.getMonth() === t.getMonth() && d.getFullYear() === t.getFullYear();
}

function formatDate(time: string): string {
  return new Date(time).toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

function ConfirmDeleteModal({ visible, onConfirm, onCancel }: {
  visible: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onCancel}>
      <Pressable style={styles.overlay} onPress={onCancel}>
        <Pressable style={styles.confirmBox}>
          <Text style={styles.confirmTitle}>Supprimer ce message</Text>
          <Text style={styles.confirmBody}>
            Conformément au RGPD, le contenu sera effacé définitivement.
          </Text>
          <View style={styles.confirmActions}>
            <TouchableOpacity style={styles.confirmBtnCancel} onPress={onCancel}>
              <Text style={styles.confirmBtnCancelText}>Annuler</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.confirmBtnDelete} onPress={onConfirm}>
              <Text style={styles.confirmBtnDeleteText}>Supprimer</Text>
            </TouchableOpacity>
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

function Bubble({ message, colors, onRequestDelete }: {
  message: Message;
  colors: typeof Colors.light;
  onRequestDelete: (id: string) => void;
}) {
  const isMe = message.from === 'me';
  const today = isToday(message.time);
  const deleted = message.text === "[contenu supprimé à la demande de l'auteur]";

  function handleDelete() {
    if (!isMe || deleted) return;
    onRequestDelete(message.id);
  }

  return (
    <View style={[styles.row, isMe ? styles.rowMe : styles.rowOther]}>
      <View style={styles.bubbleWrapper}>
        <TouchableOpacity
          activeOpacity={0.85}
          onPress={Platform.OS === 'web' ? handleDelete : undefined}
          onLongPress={Platform.OS !== 'web' ? handleDelete : undefined}
          delayLongPress={400}
        >
          <View style={[
            styles.bubble,
            isMe
              ? [styles.bubbleMe, { backgroundColor: colors.bubbleMe }]
              : [styles.bubbleOther, { backgroundColor: colors.bubbleOther }],
          ]}>
            <Text style={[
              styles.bubbleText,
              { color: colors.bubbleText },
              deleted && styles.bubbleDeleted,
            ]}>{message.text}</Text>
            <Text style={[
              styles.time,
              { color: isMe ? colors.bubbleTimeMeColor : colors.bubbleTime },
            ]}>{formatTime(message.time)}</Text>
          </View>
        </TouchableOpacity>
        {!today && (
          <Text style={[styles.dateBelow, { textAlign: isMe ? 'right' : 'left', color: colors.bubbleTime }]}>
            {formatDate(message.time)}
          </Text>
        )}
      </View>
    </View>
  );
}

export default function ChatScreen() {
  const colors = useTheme();
  const navMenu = useNavMenu('chat');
  const [messages, setMessages] = useState<Message[]>([]);
  const [months, setMonths] = useState(1);
  const [monthsOpen, setMonthsOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const errorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);

  function showError(msg: string) {
    setError(msg);
    if (errorTimer.current) clearTimeout(errorTimer.current);
    errorTimer.current = setTimeout(() => setError(null), 15_000);
  }
  const [input, setInput] = useState('');
  const listRef = useRef<any>(null);
  const { width, height } = useWindowDimensions();
  const isLandscape = width > height;
  const orientation = useOrientation();
  const [keyboardOffset, setKeyboardOffset] = useState(56);

  const safeEdges = useMemo((): ('bottom' | 'left' | 'right')[] | EdgeRecord => {
    if (Platform.OS === 'web') {
      if (orientation === 'landscape-left') return { top: 'off', bottom: 'off', left: 'additive', right: 'off' };
      if (orientation === 'landscape-right') return { top: 'off', bottom: 'off', left: 'off', right: 'additive' };
      return { top: 'off', bottom: 'off', left: 'off', right: 'off' };
    }
    if (orientation === 'landscape-left') return ['bottom', 'left'];
    if (orientation === 'landscape-right') return ['bottom', 'right'];
    return ['bottom'];
  }, [orientation]);

  useEffect(() => {
    const show = Keyboard.addListener('keyboardDidShow', () => setKeyboardOffset(90));
    const hide = Keyboard.addListener('keyboardDidHide', () => setKeyboardOffset(56));
    return () => { show.remove(); hide.remove(); };
  }, []);

  useEffect(() => {
    if (messages.length > 0) {
      const t = setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 500);
      return () => clearTimeout(t);
    }
  }, [messages.length]);

  useFocusEffect(
    useCallback(() => {
      let cancelled = false;
      const fetchHistory = async (showLoader: boolean) => {
        if (showLoader) setLoading(true);
        try {
          const pseudo = await getPseudo();
          if (!pseudo || cancelled) return;
          const res = await apiInstance.get<ChatItem[]>('/reponse/history', {
            headers: { 'X-Pseudo': encodeURIComponent(pseudo) },
            params: { months: String(months) },
          });
          if (!cancelled) {
            setMessages(res.data.map(item => ({
              id: item.id,
              text: item.text,
              from: item.source,
              time: item.ts,
            })));
          }
        } catch (e) {
          showError('Impossible de charger les messages.');
        } finally {
          if (!cancelled) setLoading(false);
        }
      };
      fetchHistory(true);
      const interval = setInterval(() => fetchHistory(false), 3 * 60_000);
      return () => { cancelled = true; clearInterval(interval); };
    }, [months])
  );

  const dynamicStyles = useMemo(() => StyleSheet.create({
    page: { backgroundColor: colors.pageBg },
    inputBar: {
      backgroundColor: colors.inputBar,
      borderTopColor: colors.inputBarBorder,
    },
    input: {
      backgroundColor: colors.inputBg,
      borderColor: colors.inputBorder,
      color: colors.inputText,
    },
  }), [colors]);

  const monthOptions = MONTH_OPTIONS.map(m => ({ value: m, label: m === 1 ? '1 mois' : `${m} mois` }));

  async function deleteMessage(messageId: string) {
    try {
      const pseudo = await getPseudo();
      if (!pseudo) return;
      await apiInstance.delete(`/feedback/${messageId}`, { headers: { 'X-Pseudo': encodeURIComponent(pseudo) } });
      setMessages(prev => prev.map(m =>
        m.id === messageId
          ? { ...m, text: "[contenu supprimé à la demande de l'auteur]" }
          : m
      ));
    } catch (e) {
      showError('Impossible de supprimer le message.');
    }
  }

  async function send() {
    const text = input.trim();
    if (!text) return;
    try {
      const pseudo = await getPseudo();
      const message_id = generateUUID();
      await apiInstance.post('/feedback', [{ content: text, pseudo, message_id }]);
      setInput('');
      setMessages(prev => [
        ...prev,
        { id: message_id, text, from: 'me', time: now() },
      ]);
    } catch (e) {
      console.log(e)
      showError("Impossible d'envoyer le message.");
    }
  }

  return (
    <>
      <Stack.Screen
        options={{
          headerRight: () => <HeaderMenu items={navMenu} />,
        }}
      />
      <SafeAreaView style={[styles.page, dynamicStyles.page]} edges={safeEdges}>
        <KeyboardAvoidingView
          key={isLandscape ? 'landscape' : 'portrait'}
          style={styles.container}
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : keyboardOffset}
        >
          <ZenBackground />

          <View style={styles.filterRow}>
            <TouchableOpacity
              style={[styles.filterBtn, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}
              onPress={() => setMonthsOpen(true)}
            >
              <Text style={[styles.filterBtnText, { color: colors.textPrimary }]}>
                {months === 1 ? '1 mois' : `${months} mois`}
              </Text>
              <Text style={[styles.filterCaret, { color: colors.textSecondary }]}>▾</Text>
            </TouchableOpacity>
          </View>

          {error && (
            <View style={styles.errorBanner}>
              <Text style={styles.errorBannerText}>{error}</Text>
            </View>
          )}

          <DropdownModal
            visible={monthsOpen}
            onClose={() => setMonthsOpen(false)}
            options={monthOptions}
            selected={months}
            onSelect={setMonths}
          />

          <ConfirmDeleteModal
            visible={pendingDeleteId !== null}
            onConfirm={() => { deleteMessage(pendingDeleteId!); setPendingDeleteId(null); }}
            onCancel={() => setPendingDeleteId(null)}
          />

          {loading
            ? <ActivityIndicator style={styles.loader} size="large" />
            : (
              <FlashList
                ref={listRef}
                data={messages}
                extraData={messages}
                keyExtractor={m => m.id}
                renderItem={({ item }) => <Bubble message={item} colors={colors} onRequestDelete={setPendingDeleteId} />}
                contentContainerStyle={styles.list}
              />
            )
          }

          <View style={[styles.inputBar, dynamicStyles.inputBar]}>
            <TextInput
              style={[styles.input, dynamicStyles.input]}
              value={input}
              onChangeText={setInput}
              placeholder="Message"
              placeholderTextColor={colors.inputPlaceholder}
              multiline
              submitBehavior="newline"
            />
            <TouchableOpacity
              style={[styles.sendBtn, !input.trim() && styles.sendBtnDisabled]}
              onPress={send}
              activeOpacity={0.7}
              disabled={!input.trim()}
            >
              <Text style={styles.sendIcon}>➤</Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </SafeAreaView>
    </>
  );
}

const styles = StyleSheet.create({
  page: {
    flex: 1,
  },
  container: {
    flex: 1,
    backgroundColor: 'transparent',
  },
  filterRow: {
    flexDirection: 'row',
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  filterBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 16,
    borderWidth: 1,
  },
  filterBtnText: { fontSize: 13, fontWeight: '500' },
  filterCaret: { fontSize: 10 },
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.3)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  dropdown: {
    backgroundColor: '#fff',
    borderRadius: 12,
    paddingVertical: 8,
    minWidth: 180,
    boxShadow: '0 4px 8px rgba(0,0,0,0.15)',
    elevation: 8,
  },
  dropdownTitle: {
    fontSize: 11,
    fontWeight: '700',
    color: '#999',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    paddingHorizontal: 16,
    paddingBottom: 6,
    borderBottomWidth: 1,
    borderBottomColor: '#f0f0f0',
    marginBottom: 4,
  },
  dropdownItem: { paddingHorizontal: 16, paddingVertical: 10 },
  dropdownItemActive: { backgroundColor: '#e8f0fe' },
  dropdownItemText: { fontSize: 14, color: '#333' },
  dropdownItemTextActive: { color: '#1976d2', fontWeight: '600' },
  loader: {
    flex: 1,
    marginTop: 40,
  },
  list: {
    flexGrow: 1,
    justifyContent: 'flex-end',
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  row: {
    marginVertical: 3,
    flexDirection: 'row',
  },
  rowMe: {
    justifyContent: 'flex-end',
  },
  rowOther: {
    justifyContent: 'flex-start',
  },
  bubble: {
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingTop: 8,
    paddingBottom: 6,
    boxShadow: '0 1px 2px rgba(0,0,0,0.08)',
    elevation: 1,
  },
  bubbleWrapper: {
    maxWidth: '75%',
  },
  bubbleMe: {
    borderBottomRightRadius: 3,
  },
  bubbleOther: {
    borderBottomLeftRadius: 3,
  },
  bubbleText: {
    fontSize: 15,
    lineHeight: 20,
  },
  time: {
    fontSize: 11,
    alignSelf: 'flex-end',
    marginTop: 2,
  },
  dateBelow: {
    fontSize: 10,
    opacity: 0.4,
    marginTop: 2,
    marginHorizontal: 4,
  },
  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderTopWidth: 1,
    gap: 8,
  },
  input: {
    flex: 1,
    borderRadius: 24,
    paddingHorizontal: 16,
    paddingVertical: Platform.OS === 'ios' ? 10 : 8,
    fontSize: 15,
    maxHeight: 120,
    borderWidth: 1,
  },
  sendBtn: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#25d366',
    justifyContent: 'center',
    alignItems: 'center',
  },
  sendBtnDisabled: {
    backgroundColor: '#b2dfdb',
  },
  sendIcon: {
    color: '#fff',
    fontSize: 18,
    marginLeft: 2,
  },
  bubbleDeleted: {
    fontStyle: 'italic',
    opacity: 0.5,
  },
  confirmBox: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 20,
    width: '80%',
    maxWidth: 320,
    boxShadow: '0 4px 8px rgba(0,0,0,0.2)',
    elevation: 8,
  },
  confirmTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1f2937',
    marginBottom: 8,
  },
  confirmBody: {
    fontSize: 13,
    color: '#6b7280',
    marginBottom: 20,
    lineHeight: 18,
  },
  confirmActions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    gap: 12,
  },
  confirmBtnCancel: {
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  confirmBtnCancelText: {
    fontSize: 14,
    color: '#6b7280',
    fontWeight: '500',
  },
  confirmBtnDelete: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    backgroundColor: '#fee2e2',
    borderRadius: 6,
  },
  confirmBtnDeleteText: {
    fontSize: 14,
    color: '#dc2626',
    fontWeight: '600',
  },
  errorBanner: {
    backgroundColor: '#dc2626',
    paddingHorizontal: 16,
    paddingVertical: 10,
    marginHorizontal: 12,
    marginBottom: 4,
    borderRadius: 8,
  },
  errorBannerText: {
    color: '#fff',
    fontSize: 13,
    fontWeight: '500',
    textAlign: 'center',
  },
});
