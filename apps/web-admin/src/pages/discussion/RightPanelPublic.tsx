import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import PushPinIcon from '@mui/icons-material/PushPin';
import type { Feedback } from '../feedback/Feedback';
import { classifiedFeedbacks } from '../analyse/shared';
import { POST_ITS, CONNECTIONS, type PostIt } from './data';

const TAG_COLORS: Record<string, string> = {
  Algo: '#ede9fe',
  Chimie: '#dcfce7',
  Examens: '#dbeafe',
  React: '#fce7f3',
  Maths: '#fef9c3',
  Infos: '#fff7ed',
};

const TAG_TEXT: Record<string, string> = {
  Algo: '#7c3aed',
  Chimie: '#166534',
  Examens: '#1e40af',
  React: '#be185d',
  Maths: '#854d0e',
  Infos: '#c2410c',
};

const BOARD_WIDTH = 900;
const BOARD_HEIGHT = 660;
const POSTIT_W = 180;
const POSTIT_H = 165;

interface Props {
  feedbacks: Feedback[];
  onPinNote: () => void;
}

export default function RightPanelPublic({ feedbacks, onPinNote }: Props) {
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [hovered, setHovered] = useState<number | null>(null);

  const categories = useMemo(() => {
    const set = new Set(classifiedFeedbacks(feedbacks).map(f => f.categorie).filter(Boolean) as string[]);
    return Array.from(set).sort();
  }, [feedbacks]);

  const visible: PostIt[] = POST_ITS.filter(p =>
    activeCategory === null || p.tag.toLowerCase() === activeCategory.toLowerCase()
  );
  const visibleIds = new Set(visible.map(p => p.id));

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Top gradient stripe */}
      <Box sx={{ height: 3, background: 'linear-gradient(90deg, #2f5ff5, #06b6d4, #f97316)', flexShrink: 0 }} />

      {/* Floating toolbar */}
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', flexWrap: 'wrap', gap: 1, pt: 2, pb: 1.5, px: 2, flexShrink: 0 }}>
        {categories.length > 0 && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
            <Typography variant="body2" color="text.secondary">Filtrer :</Typography>
            <ToggleButtonGroup
              value={activeCategory}
              exclusive
              onChange={(_, v) => setActiveCategory(v)}
              size="small"
              sx={{ flexWrap: 'wrap', gap: 0.5 }}
            >
              {categories.map(cat => (
                <ToggleButton key={cat} value={cat} sx={{ textTransform: 'none', fontSize: '0.75rem' }}>
                  {cat}
                </ToggleButton>
              ))}
            </ToggleButtonGroup>
          </Box>
        )}
        <Button
          size="small"
          variant="contained"
          startIcon={<PushPinIcon sx={{ fontSize: '0.9rem !important' }} />}
          onClick={onPinNote}
          sx={{
            borderRadius: '999px', textTransform: 'none', fontWeight: 600, fontSize: '0.8rem',
            background: 'linear-gradient(135deg, #f97316, #f59e0b)',
            '&:hover': { background: 'linear-gradient(135deg, #ea6c00, #e8930a)' },
            boxShadow: 'none',
          }}
        >
          Épingler une note
        </Button>
      </Box>

      {/* Corkboard */}
      <Box sx={{ flex: 1, overflowY: 'auto', overflowX: 'auto' }}>
        <Box sx={{
          position: 'relative',
          width: BOARD_WIDTH,
          minHeight: BOARD_HEIGHT,
          mx: 'auto',
          background: `
            radial-gradient(circle at 15% 20%, rgba(100,120,200,0.08) 0%, transparent 45%),
            radial-gradient(circle at 85% 75%, rgba(150,100,200,0.07) 0%, transparent 45%),
            radial-gradient(circle at 50% 50%, rgba(80,100,180,0.05) 0%, transparent 60%),
            #e8ebf4
          `,
          borderRadius: 2,
          overflow: 'visible',
        }}>
          {/* SVG connection lines */}
          <svg
            style={{ position: 'absolute', inset: 0, width: BOARD_WIDTH, height: BOARD_HEIGHT, pointerEvents: 'none', overflow: 'visible' }}
          >
            <defs>
              <marker id="dot-start" markerWidth="5" markerHeight="5" refX="2.5" refY="2.5">
                <circle cx="2.5" cy="2.5" r="2" fill="#9ca3af" />
              </marker>
              <marker id="dot-end" markerWidth="5" markerHeight="5" refX="2.5" refY="2.5">
                <circle cx="2.5" cy="2.5" r="2" fill="#9ca3af" />
              </marker>
            </defs>
            {CONNECTIONS.map(([fromId, toId]) => {
              if (!visibleIds.has(fromId) || !visibleIds.has(toId)) return null;
              const from = POST_ITS.find(p => p.id === fromId)!;
              const to = POST_ITS.find(p => p.id === toId)!;
              return (
                <line
                  key={`${fromId}-${toId}`}
                  x1={from.x + POSTIT_W / 2}
                  y1={from.y + POSTIT_H / 2}
                  x2={to.x + POSTIT_W / 2}
                  y2={to.y + POSTIT_H / 2}
                  stroke="#9ca3af"
                  strokeWidth="1.5"
                  strokeDasharray="5 4"
                  markerStart="url(#dot-start)"
                  markerEnd="url(#dot-end)"
                  opacity={0.7}
                />
              );
            })}
          </svg>

          {/* Post-its */}
          {visible.map(postit => (
            <Box
              key={postit.id}
              onMouseEnter={() => setHovered(postit.id)}
              onMouseLeave={() => setHovered(null)}
              sx={{
                position: 'absolute',
                left: postit.x,
                top: postit.y,
                width: POSTIT_W,
                transform: `rotate(${postit.rotation}deg) scale(${hovered === postit.id ? 1.06 : 1})`,
                transition: 'transform 0.2s ease, box-shadow 0.2s ease, z-index 0s',
                zIndex: hovered === postit.id ? 20 : 1,
                cursor: 'default',
              }}
            >
              <Paper
                elevation={hovered === postit.id ? 8 : 2}
                sx={{
                  bgcolor: postit.bgColor,
                  borderRadius: '2px',
                  overflow: 'visible',
                  ...(postit.isTeacher && {
                    border: '2px dashed #d97706',
                    bgcolor: '#fffbeb',
                  }),
                }}
              >
                {/* Pin */}
                <Box sx={{
                  position: 'absolute',
                  top: -9,
                  left: '50%',
                  transform: 'translateX(-50%)',
                  width: 18, height: 18,
                  borderRadius: '50%',
                  background: postit.isTeacher
                    ? 'radial-gradient(circle at 38% 35%, #fcd34d, #d97706)'
                    : 'radial-gradient(circle at 38% 35%, #f87171, #dc2626)',
                  boxShadow: '0 2px 4px rgba(0,0,0,0.25)',
                  zIndex: 2,
                }} />

                <Box sx={{ p: 1.5, pt: 1.75 }}>
                  {/* Author */}
                  <Typography
                    variant="caption"
                    sx={{
                      display: 'block',
                      fontWeight: 700,
                      letterSpacing: '0.05em',
                      color: postit.isTeacher ? '#92400e' : '#6b7280',
                      fontSize: '0.6rem',
                      mb: 0.75,
                    }}
                  >
                    {postit.author}
                  </Typography>

                  {/* Message */}
                  <Typography
                    variant="body2"
                    sx={{
                      fontSize: '0.78rem',
                      lineHeight: 1.45,
                      color: '#1f2937',
                      mb: 1.25,
                      display: '-webkit-box',
                      WebkitLineClamp: 4,
                      WebkitBoxOrient: 'vertical',
                      overflow: 'hidden',
                    }}
                  >
                    {postit.text}
                  </Typography>

                  {/* Footer */}
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Chip
                      label={postit.tag}
                      size="small"
                      sx={{
                        height: 17,
                        fontSize: '0.6rem',
                        fontWeight: 700,
                        bgcolor: TAG_COLORS[postit.tag] ?? '#f1f5f9',
                        color: TAG_TEXT[postit.tag] ?? '#475569',
                      }}
                    />
                    <Typography variant="caption" sx={{ fontSize: '0.62rem', color: '#9ca3af' }}>
                      {postit.timestamp}
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Box>
          ))}
        </Box>
      </Box>
    </Box>
  );
}
