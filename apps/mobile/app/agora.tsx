import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { Colors } from '@/constants/theme';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { Stack } from 'expo-router';
import { FlatList, StyleSheet, Text, View } from 'react-native';

type PostIt = {
    id: string;
    text: string;
    author: string;
    tag: string;
    color: string;
    rotation: number;
    time: string;
};

const NOTES: PostIt[] = [
    {
        id: '1',
        text: 'Penser à revoir l\'architecture des composants avant le sprint 3.',
        author: 'Marie',
        tag: 'Tech',
        color: '#fff9c4',
        rotation: -2,
        time: 'il y a 5 min',
    },
    {
        id: '2',
        text: 'La démo est jeudi matin. Préparer les slides !',
        author: 'Paul',
        tag: 'Urgent',
        color: '#f8bbd0',
        rotation: 1.5,
        time: 'il y a 12 min',
    },
    {
        id: '3',
        text: 'Idée : ajouter un mode sombre sur l\'écran d\'accueil.',
        author: 'Léa',
        tag: 'Idée',
        color: '#c8e6c9',
        rotation: -1,
        time: 'il y a 30 min',
    },
    {
        id: '4',
        text: 'Le bug sur iOS 17 est reproduit. Ticket ouvert.',
        author: 'Thomas',
        tag: 'Bug',
        color: '#ffe0b2',
        rotation: 2.5,
        time: 'il y a 1 h',
    },
    {
        id: '5',
        text: 'Réunion retro vendredi à 14h. Pensez à noter vos points.',
        author: 'Julie',
        tag: 'RH',
        color: '#e1bee7',
        rotation: -3,
        time: 'il y a 2 h',
    },
    {
        id: '6',
        text: 'Super boulot sur la v2 ! Bravo à toute l\'équipe 🎉',
        author: 'Alex',
        tag: 'Bravo',
        color: '#b3e5fc',
        rotation: 1,
        time: 'hier',
    },
    {
        id: '7',
        text: 'Reminder : mettre à jour les dépendances npm cette semaine.',
        author: 'Marie',
        tag: 'Tech',
        color: '#fff9c4',
        rotation: -1.5,
        time: 'hier',
    },
    {
        id: '8',
        text: 'Feedback client positif sur le nouveau design 👍',
        author: 'Paul',
        tag: 'Client',
        color: '#c8e6c9',
        rotation: 2,
        time: 'avant-hier',
    },
];

const TAG_COLORS: Record<string, string> = {
    Tech: '#1976d2',
    Urgent: '#d32f2f',
    Idée: '#7b1fa2',
    Bug: '#e65100',
    RH: '#388e3c',
    Bravo: '#f57f17',
    Client: '#00796b',
};

function Note({ item, colors }: { item: PostIt; colors: typeof Colors.light }) {
    const tagColor = TAG_COLORS[item.tag] ?? '#555';

    return (
        <View style={[styles.noteWrapper, { transform: [{ rotate: `${item.rotation}deg` }] }]}>
            <View style={[styles.note, { backgroundColor: item.color }]}>
                <View style={styles.pin} />
                <View style={[styles.tag, { backgroundColor: tagColor }]}>
                    <Text style={styles.tagText}>{item.tag}</Text>
                </View>
                <Text style={styles.noteText}>{item.text}</Text>
                <View style={[styles.noteFooter, { borderTopColor: colors.noteFooterBorder }]}>
                    <Text style={[styles.author, { color: colors.noteAuthor }]}>{item.author}</Text>
                    <Text style={[styles.time, { color: colors.noteTime }]}>{item.time}</Text>
                </View>
            </View>
        </View>
    );
}

export default function AgoraScreen() {
    const colors = useTheme();
    const navMenu = useNavMenu('agora');

    return (
        <View style={styles.container}>
            <ZenBackground />
            <Stack.Screen
                options={{
                    headerRight: () => <HeaderMenu items={navMenu} />,
                }}
            />
            <FlatList
                data={NOTES}
                keyExtractor={n => n.id}
                numColumns={2}
                renderItem={({ item }) => <Note item={item} colors={colors} />}
                contentContainerStyle={styles.grid}
                columnWrapperStyle={styles.row}
            />
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: 'transparent',
    },
    grid: {
        padding: 12,
        paddingBottom: 32,
    },
    row: {
        justifyContent: 'space-between',
        marginBottom: 16,
    },
    noteWrapper: {
        width: '47%',
    },
    note: {
        borderRadius: 2,
        padding: 12,
        paddingTop: 18,
        minHeight: 160,
        shadowColor: '#000',
        shadowOffset: { width: 2, height: 4 },
        shadowOpacity: 0.25,
        shadowRadius: 6,
        elevation: 6,
        justifyContent: 'space-between',
    },
    pin: {
        position: 'absolute',
        top: -7,
        alignSelf: 'center',
        width: 14,
        height: 14,
        borderRadius: 7,
        backgroundColor: '#c0392b',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.4,
        shadowRadius: 2,
        elevation: 4,
        zIndex: 10,
    },
    tag: {
        alignSelf: 'flex-start',
        borderRadius: 4,
        paddingHorizontal: 6,
        paddingVertical: 2,
        marginBottom: 8,
    },
    tagText: {
        color: '#fff',
        fontSize: 10,
        fontWeight: '700',
        letterSpacing: 0.5,
        textTransform: 'uppercase',
    },
    noteText: {
        fontSize: 13,
        color: '#333',
        lineHeight: 19,
        flex: 1,
    },
    noteFooter: {
        marginTop: 10,
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        borderTopWidth: 1,
        borderTopColor: 'rgba(0,0,0,0.08)',
        paddingTop: 6,
    },
    author: {
        fontSize: 11,
        fontWeight: '600',
        color: '#555',
    },
    time: {
        fontSize: 10,
        color: '#888',
    },
    fab: {
        position: 'absolute',
        bottom: 28,
        right: 24,
        width: 56,
        height: 56,
        borderRadius: 28,
        backgroundColor: '#25d366',
        justifyContent: 'center',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 6,
        elevation: 8,
    },
    fabIcon: {
        fontSize: 24,
    },
    headerBtnText: {
        fontSize: 14,
        color: '#1976d2',
        fontWeight: '600',
    },
    headerBtn: {
        paddingHorizontal: 8,
        paddingVertical: 4,
    },
});
