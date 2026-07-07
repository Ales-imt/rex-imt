import { HeaderMenu } from '@/components/header-menu';
import { ZenBackground } from '@/components/zen-background';
import { useNavMenu } from '@/hooks/use-nav-menu';
import { useTheme } from '@/hooks/use-theme';
import { apiInstance } from '@/services/api';
import { Stack, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ActivityIndicator, Modal, PanResponder, Pressable, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';

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
  return dateStr.replaceAll('-', '');
}

function localToday(): string {
  return fmtLocal(new Date());
}

const MONTHS_SHORT = Array.from({ length: 12 }, (_, i) =>
  new Date(2000, i, 1).toLocaleDateString('fr-FR', { month: 'short' })
);

const WEEKDAYS = ['lun', 'mar', 'mer', 'jeu', 'ven', 'sam', 'dim'];

function weekDates(mondayStr: string): string[] {
  const base = parseLocal(mondayStr);
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(base);
    d.setDate(base.getDate() + i);
    return fmtLocal(d);
  });
}

function WeekStrip({ weekStart, selected, today, eventDates, onSelectDate, onPrevWeek, onNextWeek }: {
  weekStart: string;
  selected: string;
  today: string;
  eventDates: Set<string>;
  onSelectDate: (date: string) => void;
  onPrevWeek: () => void;
  onNextWeek: () => void;
}) {
  const colors = useTheme();
  // Keep the latest callbacks in refs so the PanResponder (created once) never goes stale.
  const navRef = useRef({ onPrevWeek, onNextWeek });
  navRef.current = { onPrevWeek, onNextWeek };

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) =>
          Math.abs(g.dx) > 20 && Math.abs(g.dx) > Math.abs(g.dy) * 1.5,
        onPanResponderRelease: (_, g) => {
          if (g.dx > 50) navRef.current.onPrevWeek();
          else if (g.dx < -50) navRef.current.onNextWeek();
        },
      }),
    []
  );

  const days = weekDates(weekStart);

  return (
    <View style={styles.weekStrip} {...panResponder.panHandlers}>
      <TouchableOpacity onPress={onPrevWeek} style={styles.weekNav} hitSlop={8}>
        <Text style={[styles.weekArrow, { color: colors.tint }]}>‹</Text>
      </TouchableOpacity>
      <View style={styles.weekDays}>
        {days.map((date, i) => {
          const isSelected = date === selected;
          const isToday = date === today;
          const hasEvents = eventDates.has(date);
          const dayNum = parseLocal(date).getDate();
          return (
            <TouchableOpacity
              key={date}
              style={styles.dayCell}
              onPress={() => onSelectDate(date)}
              activeOpacity={0.7}
            >
              <Text style={[styles.dayName, { color: colors.textSecondary }]}>{WEEKDAYS[i]}</Text>
              <View style={[styles.dayNumWrap, isSelected && { backgroundColor: colors.tint }]}>
                <Text
                  style={[
                    styles.dayNum,
                    {
                      color: isSelected ? colors.background : isToday ? colors.tint : colors.text,
                      fontWeight: isSelected || isToday ? '700' : '400',
                    },
                  ]}
                >
                  {dayNum}
                </Text>
              </View>
              <View
                style={[
                  styles.dayDot,
                  { backgroundColor: hasEvents ? colors.tint : 'transparent' },
                ]}
              />
            </TouchableOpacity>
          );
        })}
      </View>
      <TouchableOpacity onPress={onNextWeek} style={styles.weekNav} hitSlop={8}>
        <Text style={[styles.weekArrow, { color: colors.tint }]}>›</Text>
      </TouchableOpacity>
    </View>
  );
}

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

function TodayButton({ onPress }: { readonly onPress: () => void }) {
  const colors = useTheme();
  return (
    <TouchableOpacity
      onPress={onPress}
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

  const eventDates = useMemo(() => new Set(Object.keys(events)), [events]);

  const dayEvents = events[selected] ?? [];

  return (
    <>
      <Stack.Screen options={{ headerRight: () => <HeaderMenu items={navMenu} /> }} />
      <View style={styles.container}>
        <ZenBackground />
        <View style={[styles.calendarCard, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
          <View style={styles.monthHeader}>
            <TouchableOpacity style={styles.monthLabel} onPress={() => setMonthPickerVisible(true)}>
              <Text style={[styles.monthLabelText, { color: colors.textPrimary }]}>{displayMonth} ▾</Text>
            </TouchableOpacity>
          </View>
          <WeekStrip
            weekStart={weekStart}
            selected={selected}
            today={today}
            eventDates={eventDates}
            onSelectDate={setSelected}
            onPrevWeek={() => setSelected(shiftWeek(weekStart, -1))}
            onNextWeek={() => setSelected(shiftWeek(weekStart, 1))}
          />
        </View>

        <ScrollView contentContainerStyle={styles.scroll}>
          <View style={styles.sectionHeader}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>
              {selected === today ? "Aujourd'hui" : selected}
            </Text>
            {selected !== today && <TodayButton onPress={() => setSelected(today)} />}
          </View>

          {loading ? (
            <ActivityIndicator color={colors.tint} style={{ marginTop: 20 }} />
          ) : error ? (
            <Text style={[styles.empty, { color: 'red' }]}>{error}</Text>
          ) : dayEvents.length === 0 ? (
            <Text style={[styles.empty, { color: colors.textSecondary }]}>Aucun cours ce jour</Text>
          ) : (
            dayEvents.map((event, i) => (
              <View key={`${event.time}-${event.title}-${i}`} style={[styles.eventRow, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
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
  monthHeader: { flexDirection: 'row', alignItems: 'center', justifyContent: 'center', paddingTop: 8, paddingBottom: 2 },
  weekArrow: { fontSize: 24, paddingHorizontal: 10 },
  monthLabel: { paddingHorizontal: 14, paddingTop: 8, paddingBottom: 2 },
  monthLabelText: { fontSize: 14, fontWeight: '600' },
  weekStrip: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 4,
    paddingTop: 4,
    paddingBottom: 10,
  },
  weekNav: { paddingHorizontal: 4, justifyContent: 'center', alignItems: 'center' },
  weekDays: { flex: 1, flexDirection: 'row', justifyContent: 'space-between' },
  dayCell: { flex: 1, alignItems: 'center', paddingVertical: 2 },
  dayName: { fontSize: 11, marginBottom: 4, textTransform: 'capitalize' },
  dayNumWrap: {
    width: 34,
    height: 34,
    borderRadius: 17,
    justifyContent: 'center',
    alignItems: 'center',
  },
  dayNum: { fontSize: 15 },
  dayDot: { width: 5, height: 5, borderRadius: 2.5, marginTop: 4 },
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
