import { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Alert from '@mui/material/Alert';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { apiInstance } from '../../services/api';
import { ENDPOINT_FEEDBACK } from '../feedback/def';
import type { Feedback } from '../feedback/Feedback';
import LeftPanel from './LeftPanel';
import RightPanelPrive from './RightPanelPrive';
import RightPanelPublic from './RightPanelPublic';
import PinDrawer from './PinDrawer';
import type { Mode } from './data';

function useFeedbacks(months: number) {
  return useQuery<Feedback[]>({
    queryKey: ['discussion-feedbacks-recent', months],
    queryFn: async () => {
      const res = await apiInstance.get<Feedback[]>(`${ENDPOINT_FEEDBACK}/recent?months=${months}`);
      return res.data;
    },
    staleTime: 60_000,
    refetchInterval: 2 * 60_000,
  });
}

export function DiscussionPanel() {
  const [mode, setMode] = useState<Mode>('prive');
  const [months, setMonths] = useState(() => {
    const saved = localStorage.getItem('discussion.months');
    return saved ? Number(saved) : 1;
  });

  const handleMonthsChange = (value: number) => {
    setMonths(value);
    localStorage.setItem('discussion.months', String(value));
  };
  const [selectedFeedbackId, setSelectedFeedbackId] = useState<number | null>(null);
  const [pinDrawerOpen, setPinDrawerOpen] = useState(false);
  const [pinFeedback, setPinFeedback] = useState<Feedback | null>(null);

  const { data: feedbacks = [], isLoading, isError } = useFeedbacks(months);

  useEffect(() => {
    if (feedbacks.length > 0 && selectedFeedbackId === null) {
      setSelectedFeedbackId(feedbacks[0].id);
    }
  }, [feedbacks, selectedFeedbackId]);

  const selectedFeedback = feedbacks.find(f => f.id === selectedFeedbackId) ?? null;

  const handleOpenPinDrawer = (fb: Feedback) => {
    setPinFeedback(fb);
    setPinDrawerOpen(true);
  };

  const handlePin = (_text: string, _color: string) => {
    setPinDrawerOpen(false);
  };

  if (isError) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Impossible de charger les feedbacks. Vérifiez votre connexion.</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      {/* Left panel */}
      <LeftPanel
        feedbacks={feedbacks}
        isLoading={isLoading}
        months={months}
        onMonthsChange={handleMonthsChange}
        selectedFeedbackId={selectedFeedbackId}
        onSelectFeedback={setSelectedFeedbackId}
      />

      {/* Right side */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, overflow: 'hidden' }}>
        {/* Mode toggle */}
        <Box sx={{
          display: 'flex', justifyContent: 'center', alignItems: 'center',
          py: 1.5, borderBottom: '1px solid', borderColor: 'divider',
          bgcolor: 'background.paper', flexShrink: 0,
        }}>
          <Box sx={{ display: 'inline-flex', bgcolor: 'action.hover', borderRadius: '999px', p: '4px', gap: '4px' }}>
            <ModeButton
              label="🔒 Privé"
              active={mode === 'prive'}
              activeStyle={{ bgcolor: '#2f5ff5', color: '#fff', '&:hover': { bgcolor: '#1e4ed8' } }}
              onClick={() => setMode('prive')}
            />
            <ModeButton
              label="📌 Mur public"
              active={mode === 'public'}
              activeStyle={{
                background: 'linear-gradient(135deg, #f97316, #f59e0b)',
                color: '#fff',
                '&:hover': { background: 'linear-gradient(135deg, #ea6c00, #e8930a)' },
              }}
              onClick={() => setMode('public')}
            />
          </Box>
        </Box>

        {/* Panel with fade on mode change */}
        <Box
          key={mode}
          sx={{
            flex: 1, overflow: 'hidden',
            '@keyframes fadeSlideIn': {
              from: { opacity: 0, transform: 'translateY(6px)' },
              to: { opacity: 1, transform: 'translateY(0)' },
            },
            animation: 'fadeSlideIn 0.22s ease',
          }}
        >
          {mode === 'prive' ? (
            <RightPanelPrive feedback={selectedFeedback} allFeedbacks={feedbacks} />
          ) : (
            <RightPanelPublic feedbacks={feedbacks} onPinNote={() => selectedFeedback && handleOpenPinDrawer(selectedFeedback)} />
          )}
        </Box>
      </Box>

      <PinDrawer
        open={pinDrawerOpen}
        onClose={() => setPinDrawerOpen(false)}
        onPin={handlePin}
        feedback={pinFeedback}
      />
    </Box>
  );
}

interface ModeButtonProps {
  label: string;
  active: boolean;
  activeStyle: object;
  onClick: () => void;
}

function ModeButton({ label, active, activeStyle, onClick }: ModeButtonProps) {
  return (
    <Box
      onClick={onClick}
      sx={{
        px: 2.25, py: 0.75, borderRadius: '999px', cursor: 'pointer',
        fontWeight: 600, fontSize: '0.85rem', lineHeight: 1,
        transition: 'all 0.18s ease', userSelect: 'none',
        display: 'flex', alignItems: 'center', gap: 0.5,
        color: 'text.secondary',
        ...(active ? activeStyle : {}),
      }}
    >
      <Typography component="span" sx={{ fontSize: 'inherit', fontWeight: 'inherit', color: 'inherit' }}>
        {label}
      </Typography>
    </Box>
  );
}
