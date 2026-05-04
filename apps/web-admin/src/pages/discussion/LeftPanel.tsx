import { useState } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import Collapse from '@mui/material/Collapse';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import type { Feedback } from '../feedback/Feedback';
import { urgenceColor, relativeTime, SENTIMENT_LABEL, SENTIMENT_COLOR } from '../analyse/shared';

const MONTH_OPTIONS = [
  { value: 1, label: '1 mois' },
  { value: 2, label: '2 mois' },
  { value: 3, label: '3 mois' },
  { value: 6, label: '6 mois' },
  { value: 12, label: '1 an' },
];

interface LeftPanelProps {
  feedbacks: Feedback[];
  isLoading: boolean;
  months: number;
  onMonthsChange: (months: number) => void;
  selectedFeedbackId: number | null;
  onSelectFeedback: (id: number) => void;
}

export default function LeftPanel({
  feedbacks,
  isLoading,
  months,
  onMonthsChange,
  selectedFeedbackId,
  onSelectFeedback,
}: LeftPanelProps) {
  const [expandedId, setExpandedId] = useState<number | null>(null);

  const handleCardClick = (fb: Feedback) => {
    onSelectFeedback(fb.id);
    setExpandedId(prev => prev === fb.id ? null : fb.id);
  };

  return (
    <Box sx={{
      width: 300,
      flexShrink: 0,
      display: 'flex',
      flexDirection: 'column',
      borderRight: '1px solid',
      borderColor: 'divider',
      bgcolor: 'background.paper',
      overflow: 'hidden',
    }}>
      {/* Header */}
      <Box sx={{ px: 2.5, pt: 2.5, pb: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
          <Typography variant="h6" fontWeight={700} sx={{ lineHeight: 1 }}>Messages</Typography>

        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
          <Typography variant="caption" color="text.secondary">
            {isLoading ? 'Chargement…' : `${feedbacks.length} feedback${feedbacks.length > 1 ? 's' : ''} étudiant${feedbacks.length > 1 ? 's' : ''}`}
          </Typography>
          <Select
            value={months}
            onChange={e => onMonthsChange(Number(e.target.value))}
            variant="standard"
            disableUnderline
            sx={{
              fontSize: '0.72rem',
              color: 'text.secondary',
              '& .MuiSelect-select': { py: 0, pr: '20px !important' },
              '& .MuiSelect-icon': { fontSize: '1rem', top: 'calc(50% - 8px)' },
            }}
          >
            {MONTH_OPTIONS.map(o => (
              <MenuItem key={o.value} value={o.value} sx={{ fontSize: '0.8rem' }}>
                {o.label}
              </MenuItem>
            ))}
          </Select>
        </Box>
      </Box>

      {/* Scrollable feedback list */}
      <Box sx={{ flex: 1, overflowY: 'auto', py: 1 }}>
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <Box key={i} sx={{ mx: 1.5, mb: 1 }}>
              <Skeleton variant="rounded" height={80} sx={{ borderRadius: 1 }} />
            </Box>
          ))
        ) : feedbacks.length === 0 ? (
          <Typography variant="body2" color="text.secondary" textAlign="center" py={4}>
            Aucun feedback disponible.
          </Typography>
        ) : (
          feedbacks.map(fb => {
            const color = urgenceColor(fb.urgence);
            const preview = fb.content?.length > 70 ? `${fb.content.slice(0, 70)}…` : fb.content;
            const isSelected = selectedFeedbackId === fb.id;
            const sentimentColor = fb.sentiment ? SENTIMENT_COLOR[fb.sentiment] : undefined;

            return (
              <Box key={fb.id}>
                <Paper
                  elevation={0}
                  onClick={() => handleCardClick(fb)}
                  sx={{
                    mx: 1.5,
                    mb: 0.5,
                    p: 1.5,
                    cursor: 'pointer',
                    borderRadius: 1,
                    borderLeft: `4px solid ${color}`,
                    border: `1px solid`,
                    borderColor: isSelected ? 'primary.main' : 'divider',
                    borderLeftColor: color,
                    bgcolor: isSelected ? 'primary.50' : 'transparent',
                    transition: 'all 0.15s',
                    '&:hover': {
                      bgcolor: isSelected ? 'primary.50' : 'action.hover',
                    },
                  }}
                >
                  {fb.resume && (
                    <Typography variant="body2" fontWeight={600} sx={{ mb: 0.25 }} noWrap>
                      {fb.resume}
                    </Typography>
                  )}
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                    {preview}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 0.5, mt: 0.75, flexWrap: 'wrap', alignItems: 'center' }}>
                    {fb.categorie && (
                      <Chip label={fb.categorie} size="small" variant="outlined" sx={{ height: 18, fontSize: '0.65rem' }} />
                    )}
                    {fb.promotion && (
                      <Chip label={fb.promotion} size="small" variant="outlined" color="primary" sx={{ height: 18, fontSize: '0.65rem' }} />
                    )}
                    {fb.urgence != null && (
                      <Chip
                        label={`U${fb.urgence}`}
                        size="small"
                        sx={{
                          height: 18,
                          fontSize: '0.65rem',
                          fontWeight: 700,
                          bgcolor: color,
                          color: fb.urgence <= 1 ? '#555' : '#fff',
                        }}
                      />
                    )}
                    {fb.sentiment && sentimentColor && (
                      <Chip
                        label={SENTIMENT_LABEL[fb.sentiment] ?? fb.sentiment}
                        size="small"
                        sx={{
                          height: 18,
                          fontSize: '0.65rem',
                          bgcolor: sentimentColor + '22',
                          color: sentimentColor,
                          fontWeight: 600,
                        }}
                      />
                    )}
                    <Typography variant="caption" color="text.disabled" sx={{ ml: 'auto', fontSize: '0.62rem' }}>
                      {relativeTime(fb.created_at)}
                    </Typography>
                  </Box>
                </Paper>

                {/* Expanded: full content */}
                <Collapse in={expandedId === fb.id} timeout={200}>
                  <Box sx={{
                    mx: 1.5, mb: 1, px: 1.5, py: 1,
                    borderLeft: `3px solid ${color}`,
                    bgcolor: 'action.hover',
                    borderRadius: '0 4px 4px 0',
                  }}>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                      {fb.content}
                    </Typography>
                  </Box>
                </Collapse>
              </Box>
            );
          })
        )}
      </Box>

    </Box>
  );
}
