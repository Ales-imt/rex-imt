import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { Stack } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';


export const NotesScreen = () => {
    const colors = useTheme();
    const navMenu = useNavMenu('notes');
    return (
        <>

            <Stack.Screen
                options={{
                    headerRight: () => <HeaderMenu items={navMenu} />,
                }}
            />
            <View style={styles.container}>
                <ZenBackground />
                <Text style={[styles.title, { color: colors.text }]}>À propos</Text>
            </View>


        </>

    )
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: 'transparent',
        justifyContent: 'center',
        alignItems: 'center',
    },
    title: {
        fontSize: 24,
        fontWeight: '600',
    },
});

export default NotesScreen;