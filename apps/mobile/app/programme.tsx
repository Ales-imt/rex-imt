import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { Stack } from 'expo-router';
import { useContext, useState } from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { CalendarProvider, WeekCalendar } from 'react-native-calendars';
import { UpdateSources } from 'react-native-calendars/src/expandableCalendar/commons';
import CalendarContext from 'react-native-calendars/src/expandableCalendar/Context';
import { MarkedDates } from 'react-native-calendars/src/types';

type Event = {
  title: string;
  time: string;
};

type EventMap = Record<string, Event[]>;

const EVENTS: EventMap = {
  [new Date().toISOString().split('T')[0]]: [
    { title: 'Réunion équipe', time: '09:00' },
    { title: 'Démo produit', time: '14:30' },
  ],
};

const PRIMARY = '#1976d2';

function TodayButton({ today }: { today: string }) {
  const { setDate } = useContext(CalendarContext);
  return (
    <TouchableOpacity onPress={() => setDate(today, UpdateSources.TODAY_PRESS)} style={styles.todayBtn}>
      <Text style={styles.todayBtnText}>Aujourd'hui</Text>
    </TouchableOpacity>
  );
}

export const ProgrammeScreen = () => {
  const colors = useTheme();
  const navMenu = useNavMenu('programme');
  const today = new Date().toISOString().split('T')[0];
  const [selected, setSelected] = useState(today);

  const markedDates: MarkedDates = Object.keys(EVENTS).reduce<MarkedDates>((acc, date) => {
    acc[date] = { marked: true, dotColor: PRIMARY };
    return acc;
  }, {});

  const calendarTheme = {
    backgroundColor: 'transparent',
    calendarBackground: 'transparent',
    textSectionTitleColor: colors.textSecondary,
    selectedDayBackgroundColor: PRIMARY,
    selectedDayTextColor: '#ffffff',
    todayTextColor: PRIMARY,
    dayTextColor: colors.text,
    textDisabledColor: colors.textSecondary,
    dotColor: PRIMARY,
    selectedDotColor: '#ffffff',
    arrowColor: PRIMARY,
    monthTextColor: colors.text,
    indicatorColor: PRIMARY,
  };

  const dayEvents = EVENTS[selected] ?? [];

  return (
    <>
      <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />
      <View style={styles.container}>
        <ZenBackground />
        <CalendarProvider date={today} onDateChanged={setSelected}>
          <View style={[styles.calendarCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <WeekCalendar
              markedDates={markedDates}
              theme={calendarTheme}
              allowShadow={false}
            />
          </View>

          <ScrollView contentContainerStyle={styles.scroll}>
            <View style={styles.sectionHeader}>
              <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>
                {selected === today ? "Aujourd'hui" : selected}
              </Text>
              {selected !== today && <TodayButton today={today} />}
            </View>
            {dayEvents.length === 0 ? (
              <Text style={[styles.empty, { color: colors.textSecondary }]}>Aucun événement</Text>
            ) : (
              dayEvents.map((event, i) => (
                <View key={i} style={[styles.eventRow, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                  <View style={styles.eventTime}>
                    <Text style={[styles.eventTimeText, { color: PRIMARY }]}>{event.time}</Text>
                  </View>
                  <Text style={[styles.eventTitle, { color: colors.textPrimary }]}>{event.title}</Text>
                </View>
              ))
            )}
          </ScrollView>
        </CalendarProvider>
      </View>
    </>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  calendarCard: {
    borderBottomWidth: 1,
  },
  scroll: {
    padding: 16,
    gap: 12,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 4,
  },
  empty: {
    fontSize: 14,
    fontStyle: 'italic',
  },
  eventRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 12,
    gap: 12,
  },
  eventTime: {
    width: 44,
  },
  eventTimeText: {
    fontSize: 13,
    fontWeight: '600',
  },
  eventTitle: {
    fontSize: 15,
    flex: 1,
  },
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
  },
  todayBtn: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
    backgroundColor: PRIMARY,
  },
  todayBtnText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '600',
  },
});

export default ProgrammeScreen;
