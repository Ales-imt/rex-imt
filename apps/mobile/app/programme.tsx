import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { Stack, useFocusEffect } from 'expo-router';
import { useCallback, useContext, useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Modal, Platform, Pressable, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { CalendarProvider, WeekCalendar } from 'react-native-calendars';
import { UpdateSources } from 'react-native-calendars/src/expandableCalendar/commons';
import CalendarContext from 'react-native-calendars/src/expandableCalendar/Context';
import { MarkedDates } from 'react-native-calendars/src/types';

type ApiCours = {
  date: string;   // YYYY-MM-DD
  hd: string;     // HH:MM
  hf: string;     // HH:MM
  cours: string;
  salle: string;
  prof: string;
  promo: string;
};

type Event = {
  title: string;
  time: string;
  endTime: string;
  room: string;
  prof: string;
};

type EventMap = Record<string, Event[]>;

let _lastSelected: string | null = null;

function toEventMap(data: ApiCours[]): EventMap {
  const map: EventMap = {};
  for (const c of data) {
    if (!map[c.date]) map[c.date] = [];
    map[c.date].push({
      title: c.cours,
      time: c.hd,
      endTime: c.hf,
      room: c.salle,
      prof: c.prof,
    });
    map[c.date].sort((a, b) => a.time.localeCompare(b.time));
  }
  return map;
}

function parseLocal(dateStr: string): Date {
  const [y, m, d] = dateStr.split('-').map(Number);
  return new Date(y, m - 1, d);
}

function fmtLocal(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function getMonday(dateStr: string): string {
  const d = parseLocal(dateStr);
  const day = d.getDay(); // 0=Sun, 1=Mon, ...
  d.setDate(d.getDate() + (day === 0 ? -6 : 1 - day));
  return fmtLocal(d);
}

function getSunday(mondayStr: string): string {
  const d = parseLocal(mondayStr);
  d.setDate(d.getDate() + 6);
  return fmtLocal(d);
}

function shiftWeek(mondayStr: string, delta: number): string {
  const d = parseLocal(mondayStr);
  d.setDate(d.getDate() + delta * 7);
  return fmtLocal(d);
}

function toApiDate(dateStr: string): string {
  return dateStr.replace(/-/g, '');
}

function localToday(): string {
  return fmtLocal(new Date());
}

const MONTHS_SHORT = Array.from({ length: 12 }, (_, i) =>
  new Date(2000, i, 1).toLocaleDateString('fr-FR', { month: 'short' })
);

function MonthPicker({ visible, onClose, onSelect, selectedYear, selectedMonth }: {
  visible: boolean;
  onClose: () => void;
  onSelect: (year: number, month: number) => void;
  selectedYear: number;
  selectedMonth: number;
}) {
  const colors = useTheme();
  const [year, setYear] = useState(selectedYear);
  useEffect(() => { if (visible) setYear(selectedYear); }, [visible]);

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={styles.modalOverlay} onPress={onClose}>
        <Pressable style={[styles.monthPickerBox, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
          <View style={styles.yearRow}>
            <TouchableOpacity onPress={() => setYear(y => y - 1)}>
              <Text style={[styles.yearArrow, { color: colors.tint }]}>‹</Text>
            </TouchableOpacity>
            <Text style={[styles.yearText, { color: colors.textPrimary }]}>{year}</Text>
            <TouchableOpacity onPress={() => setYear(y => y + 1)}>
              <Text style={[styles.yearArrow, { color: colors.tint }]}>›</Text>
            </TouchableOpacity>
          </View>
          {Array.from({ length: 4 }, (_, row) => (
            <View key={row} style={styles.monthRow}>
              {MONTHS_SHORT.slice(row * 3, row * 3 + 3).map((name, col) => {
                const i = row * 3 + col;
                const active = year === selectedYear && i === selectedMonth;
                return (
                  <TouchableOpacity
                    key={i}
                    style={[styles.monthCell, active && { backgroundColor: colors.tint }]}
                    onPress={() => { onSelect(year, i); onClose(); }}
                  >
                    <Text style={[styles.monthCellText, { color: active ? colors.background : colors.textPrimary }]}>
                      {name}
                    </Text>
                  </TouchableOpacity>
                );
              })}
            </View>
          ))}
        </Pressable>
      </Pressable>
    </Modal>
  );
}

function TodayButton({ today }: { today: string }) {
  const { setDate } = useContext(CalendarContext);
  const colors = useTheme();
  return (
    <TouchableOpacity
      onPress={() => setDate(today, UpdateSources.TODAY_PRESS)}
      style={[styles.todayBtn, { backgroundColor: colors.tint }]}
    >
      <Text style={[styles.todayBtnText, { color: colors.background }]}>Aujourd'hui</Text>
    </TouchableOpacity>
  );
}

export const ProgrammeScreen = () => {
  const colors = useTheme();
  const navMenu = useNavMenu('programme');
  const today = localToday();
  const [selected, setSelected] = useState(() => _lastSelected ?? today);
  const [weekStart, setWeekStart] = useState(() => getMonday(_lastSelected ?? today));
  const [events, setEvents] = useState<EventMap>({});
  const eventsRef = useRef(events);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshKey, setRefreshKey] = useState(0);
  const [monthPickerVisible, setMonthPickerVisible] = useState(false);
  const initialMount = useRef(true);
  const weekStartRef = useRef(weekStart);

  const selDate = parseLocal(selected);
  const displayMonth = (() => {
    const s = selDate.toLocaleDateString('fr-FR', { month: 'long', year: 'numeric' });
    return s.charAt(0).toUpperCase() + s.slice(1);
  })();

  const handleMonthSelect = (year: number, month: number) => {
    const p = (n: number) => String(n).padStart(2, '0');
    setSelected(`${year}-${p(month + 1)}-01`);
  };

  useFocusEffect(
    useCallback(() => {
      const id = setInterval(() => setRefreshKey(k => k + 1), 10_000);
      return () => clearInterval(id);
    }, [])
  );

  useEffect(() => {
    _lastSelected = selected;
    const newWeekStart = getMonday(selected);
    if (newWeekStart !== weekStart) {
      setWeekStart(newWeekStart);
    }
  }, [selected]);

  useEffect(() => {
    if (initialMount.current) { initialMount.current = false; return; }
    setSelected(weekStart);
  }, [weekStart]);

  useEffect(() => {
    const weekChanged = weekStart !== weekStartRef.current;
    weekStartRef.current = weekStart;
    if (weekChanged) { setLoading(true); setError(''); }
    const start = toApiDate(weekStart);
    const end = toApiDate(getSunday(weekStart));
    let active = true;
    apiInstance
      .get<ApiCours[]>(`/programme?start=${start}&end=${end}`)
      .then(res => {
        if (!active) return;
        const newMap = toEventMap(res.data ?? []);
        if (JSON.stringify(newMap) !== JSON.stringify(eventsRef.current)) {
          eventsRef.current = newMap;
          setEvents(newMap);
        }
      })
      .catch(() => { if (active) setError('Impossible de charger le programme.'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [weekStart, refreshKey]);

  const markedDates: MarkedDates = Object.keys(events).reduce<MarkedDates>((acc, date) => {
    acc[date] = { marked: true, dotColor: colors.tint };
    return acc;
  }, {});

  const calendarTheme = {
    backgroundColor: 'transparent',
    calendarBackground: 'transparent',
    textSectionTitleColor: colors.textSecondary,
    selectedDayBackgroundColor: colors.tint,
    selectedDayTextColor: colors.background,
    todayTextColor: colors.tint,
    dayTextColor: colors.text,
    textDisabledColor: colors.textSecondary,
    dotColor: colors.tint,
    selectedDotColor: colors.background,
    arrowColor: colors.tint,
    monthTextColor: colors.text,
    indicatorColor: colors.tint,
  };

  const dayEvents = events[selected] ?? [];

  return (
    <>
      <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />
      <View style={styles.container}>
        <ZenBackground />
        <CalendarProvider date={selected} onDateChanged={setSelected}>
          <View style={[styles.calendarCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <View style={styles.monthHeader}>
              {Platform.OS === 'web' && (
                <TouchableOpacity onPress={() => setWeekStart(w => shiftWeek(w, -1))}>
                  <Text style={[styles.weekArrow, { color: colors.tint }]}>‹</Text>
                </TouchableOpacity>
              )}
              <TouchableOpacity style={styles.monthLabel} onPress={() => setMonthPickerVisible(true)}>
                <Text style={[styles.monthLabelText, { color: colors.textPrimary }]}>{displayMonth} ▾</Text>
              </TouchableOpacity>
              {Platform.OS === 'web' && (
                <TouchableOpacity onPress={() => setWeekStart(w => shiftWeek(w, 1))}>
                  <Text style={[styles.weekArrow, { color: colors.tint }]}>›</Text>
                </TouchableOpacity>
              )}
            </View>
            <WeekCalendar
              markedDates={markedDates}
              theme={calendarTheme}
              allowShadow={false}
              firstDay={1}
            />
          </View>

          <ScrollView contentContainerStyle={styles.scroll}>
            <View style={styles.sectionHeader}>
              <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>
                {selected === today ? "Aujourd'hui" : selected}
              </Text>
              {selected !== today && <TodayButton today={today} />}
            </View>

            {loading ? (
              <ActivityIndicator color={colors.tint} style={{ marginTop: 20 }} />
            ) : error ? (
              <Text style={[styles.empty, { color: 'red' }]}>{error}</Text>
            ) : dayEvents.length === 0 ? (
              <Text style={[styles.empty, { color: colors.textSecondary }]}>Aucun cours ce jour</Text>
            ) : (
              dayEvents.map((event, i) => (
                <View key={i} style={[styles.eventRow, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
                  <View style={styles.eventTime}>
                    <Text style={[styles.eventTimeText, { color: colors.tint }]}>{event.time}</Text>
                    <Text style={[styles.eventTimeEnd, { color: colors.textSecondary }]}>{event.endTime}</Text>
                  </View>
                  <View style={styles.eventBody}>
                    <Text style={[styles.eventTitle, { color: colors.textPrimary }]} numberOfLines={2}>
                      {event.title}
                    </Text>
                    <Text style={[styles.eventMeta, { color: colors.textSecondary }]}>
                      {[event.room, event.prof].filter(Boolean).join(' · ')}
                    </Text>
                  </View>
                </View>
              ))
            )}
          </ScrollView>
        </CalendarProvider>
        <MonthPicker
          visible={monthPickerVisible}
          onClose={() => setMonthPickerVisible(false)}
          onSelect={handleMonthSelect}
          selectedYear={selDate.getFullYear()}
          selectedMonth={selDate.getMonth()}
        />
      </View>
    </>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1 },
  calendarCard: { borderBottomWidth: 1 },
  scroll: { padding: 16, gap: 12 },
  sectionTitle: { fontSize: 16, fontWeight: '600', marginBottom: 4 },
  empty: { fontSize: 14, fontStyle: 'italic' },
  eventRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 12,
    gap: 12,
  },
  eventTime: { width: 44 },
  eventTimeText: { fontSize: 13, fontWeight: '600' },
  eventTimeEnd: { fontSize: 11, marginTop: 2 },
  eventBody: { flex: 1 },
  eventTitle: { fontSize: 15, fontWeight: '500', marginBottom: 2 },
  eventMeta: { fontSize: 12 },
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
  },
  todayBtnText: { fontSize: 12, fontWeight: '600' },
  monthHeader: { flexDirection: 'row', alignItems: 'center', paddingTop: 8, paddingBottom: 2 },
  weekArrow: { fontSize: 24, paddingHorizontal: 10 },
  monthLabel: { paddingHorizontal: 14, paddingTop: 8, paddingBottom: 2 },
  monthLabelText: { fontSize: 14, fontWeight: '600' },
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  monthPickerBox: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 16,
    width: 280,
  },
  yearRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  yearArrow: { fontSize: 24, paddingHorizontal: 8 },
  yearText: { fontSize: 16, fontWeight: '600' },
  monthRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 8 },
  monthCell: { flex: 1, marginHorizontal: 3, paddingVertical: 8, borderRadius: 6, alignItems: 'center' },
  monthCellText: { fontSize: 13 },
});

export default ProgrammeScreen;
