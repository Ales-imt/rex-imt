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

const INITIAL_MESSAGES: Message[] = [
  { id: '1', text: 'Salut ! Comment ça va ?', from: 'other', time: '09:41' },
  { id: '2', text: 'Très bien merci, et toi ?', from: 'me', time: '09:42' },
  { id: '3', text: 'Tu travailles sur quoi en ce moment ?', from: 'other', time: '09:42' },
  { id: '4', text: 'Une app React Native 🚀', from: 'me', time: '09:43' },
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

let messageCache: Message[] = INITIAL_MESSAGES;

function now() {
  return new Date().toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
}

function Bubble({ message, colors }: { message: Message; colors: typeof Colors.light }) {
  const isMe = message.from === 'me';
  return (
    <View style={[styles.row, isMe ? styles.rowMe : styles.rowOther]}>
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
        ]}>{message.time}</Text>
      </View>
    </View>
  );
}

const formatter = new Intl.DateTimeFormat('fr-FR', {
  hour: '2-digit',
  minute: '2-digit',
  day: '2-digit',
  month: 'long',
  year: 'numeric'
});

export default function ChatScreen() {
  const colors = useTheme();
  const navMenu = useNavMenu('chat');
  const [messages, setMessages] = useState<Message[]>(messageCache);

  useEffect(() => {
    loadMessages().then(msgs => {
      messageCache = msgs;
      setMessages(msgs);
    });
  }, []);

  function updateMessages(updater: (prev: Message[]) => Message[]) {
    setMessages(prev => {
      const next = updater(prev);
      messageCache = next;
      saveMessages(next);
      return next;
    });
  }
  const [input, setInput] = useState('');
  const listRef = useRef<FlatList>(null);
  const { width, height } = useWindowDimensions();
  const isLandscape = width > height;
  const orientation = useOrientation();
  const [keyboardOffset, setKeyboardOffset] = useState(56);

  console.log('ChatScreen orientation', orientation);


  const safeEdges = useMemo((): ('bottom' | 'left' | 'right')[] => {
    if (orientation === 'landscape-left')  return ['bottom', 'left'];
    if (orientation === 'landscape-right') return ['bottom', 'right'];
    return ['bottom'];
  }, [orientation]);

  useEffect(() => {
    const show = Keyboard.addListener('keyboardDidShow', () => setKeyboardOffset(90));
    const hide = Keyboard.addListener('keyboardDidHide', () => setKeyboardOffset(56));
    return () => { show.remove(); hide.remove(); };
  }, []);

  useEffect(() => {
    const poll = async () => {
      try {
        const pseudo = await getPseudo();
        if (!pseudo) return;
        const res = await apiInstance.get<ReponseItem[]>('/reponse', { headers: { 'X-Pseudo': pseudo } });
        if (!res.data) return;
        if (res.data.length === 0) return;
        updateMessages(prev => [
          ...prev,
          ...res.data.map(r => ({
            id: String(r.id),
            text: r.contenu,
            from: 'other' as const,
            time: r.cree_le,
          })),
        ]);
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
    setInput('');
    updateMessages(prev => [
      ...prev,
      { id: Date.now().toString(), text, from: 'me', time: now() },
    ]);
    try {
      const pseudo = await getPseudo();
      const message_id = generateUUID();
      await apiInstance.post('/feedback', [{ content: text, pseudo, message_id }]);
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
          <FlatList
            ref={listRef}
            data={messages}
            keyExtractor={m => m.id}
            renderItem={({ item }) => <Bubble message={item} colors={colors} />}
            contentContainerStyle={styles.list}
            onContentSizeChange={() => listRef.current?.scrollToEnd({ animated: true })}
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
    maxWidth: '75%',
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
