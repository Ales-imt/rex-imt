import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { Colors } from '@/constants/theme';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useOrientation } from '@/hooks/use-orientation';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { generateUUID, getPseudo } from '@/services/tokens';
import * as FileSystem from 'expo-file-system/legacy';
import { Stack } from 'expo-router';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  FlatList,
  Keyboard,
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  useWindowDimensions,
  View,
} from 'react-native';
import { FlashList } from "@shopify/flash-list";
import { SafeAreaView } from 'react-native-safe-area-context';


type Message = {
  id: string;
  text: string;
  from: 'me' | 'other';
  time: string;
};

type ReponseItem = {
  id: number;
  contenu: string;
  auteur: string;
  cree_le: string;
};

const yesterday = (() => {
  const d = new Date();
  d.setDate(d.getDate() - 10);
  d.setHours(1, 0, 0, 0);
  return d.toISOString();
})();

const INITIAL_MESSAGES: Message[] = [
];

const CHAT_FILE = `${FileSystem.documentDirectory}chat_messages.json`;

async function loadMessages(): Promise<Message[]> {
  try {
    const info = await FileSystem.getInfoAsync(CHAT_FILE);
    if (!info.exists) return INITIAL_MESSAGES;
    const json = await FileSystem.readAsStringAsync(CHAT_FILE);
    return JSON.parse(json) as Message[];
  } catch {
    return INITIAL_MESSAGES;
  }
}

async function saveMessages(msgs: Message[]): Promise<void> {
  await FileSystem.writeAsStringAsync(CHAT_FILE, JSON.stringify(msgs));
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

function Bubble({ message, colors }: { message: Message; colors: typeof Colors.light }) {
  const isMe = message.from === 'me';
  const today = isToday(message.time);
  return (
    <View style={[styles.row, isMe ? styles.rowMe : styles.rowOther]}>
      <View style={styles.bubbleWrapper}>
        <View style={[
          styles.bubble,
          isMe
            ? [styles.bubbleMe, { backgroundColor: colors.bubbleMe }]
            : [styles.bubbleOther, { backgroundColor: colors.bubbleOther }],
        ]}>
          <Text style={[styles.bubbleText, { color: colors.bubbleText }]}>{message.text}</Text>
          <Text style={[
            styles.time,
            { color: isMe ? colors.bubbleTimeMeColor : colors.bubbleTime },
          ]}>{formatTime(message.time)}</Text>
        </View>
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

  useEffect(() => {
    loadMessages().then(msgs => {
      setMessages(msgs);
    });
  }, []);

  function updateMessages(updater: (prev: Message[]) => Message[]) {
    setMessages(prev => {
      const next = updater(prev);
      saveMessages(next);
      return next;
    });
  }
  const [input, setInput] = useState('');
  const listRef = useRef<any>(null);
  const { width, height } = useWindowDimensions();
  const isLandscape = width > height;
  const orientation = useOrientation();
  const [keyboardOffset, setKeyboardOffset] = useState(56);

  console.log('ChatScreen messages 2', messages);


  const safeEdges = useMemo((): ('bottom' | 'left' | 'right')[] => {
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
      // evite d'avoir des pertes de message.
      const t = setTimeout(() => listRef.current?.scrollToEnd({ animated: true }), 500);
      return () => clearTimeout(t);
    }
  }, [messages.length]);

  useEffect(() => {
    const poll = async () => {
      try {
        const pseudo = await getPseudo();
        if (!pseudo) return;
        const res = await apiInstance.get<ReponseItem[]>('/reponse', { headers: { 'X-Pseudo': pseudo } });
        if (!res.data) return;
        if (res.data.length === 0) return;
        updateMessages(prev => {
          const existingIds = new Set(prev.map(m => m.id));
          const newOnes = res.data
            .map(r => ({
              id: `reponse-${r.id}`,
              text: r.contenu,
              from: 'other' as const,
              time: r.cree_le,
            }))
            .filter(m => !existingIds.has(m.id));
          return newOnes.length > 0 ? [...prev, ...newOnes] : prev;
        });
      } catch (e) {
        console.error('Erreur poll réponses', e);
      }
    };

    poll();
    const id = setInterval(poll, 10_000);
    return () => clearInterval(id);
  }, []);

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

  async function send() {
    const text = input.trim();
    if (!text) return;

    try {
      const pseudo = await getPseudo();
      const message_id = generateUUID();
      await apiInstance.post('/feedback', [{ content: text, pseudo, message_id }]);
      setInput('');
      updateMessages(prev => [
        ...prev,
        { id: Date.now().toString(), text, from: 'me', time: now() },
      ])
    } catch (e) {
      console.error('Erreur envoi feedback', e);
    }
  }

  return (
    <>
      <Stack.Screen
        options={{
          headerRight: () => (
            <HeaderMenu items={navMenu} />
          ),
        }}
      />

      <SafeAreaView style={[styles.page, dynamicStyles.page]} edges={safeEdges}>

        <KeyboardAvoidingView
          key={isLandscape ? 'landscape' : 'portrait'} // Force le re-render
          style={styles.container}
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : keyboardOffset}
        >
          <ZenBackground />
          <FlashList
            ref={listRef}
            data={messages}
            extraData={messages}
            keyExtractor={m => m.id}
            renderItem={({ item }) => <Bubble message={item} colors={colors} />}
            contentContainerStyle={styles.list}
          />

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
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 2,
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
});
