import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import CircularProgress from '@mui/material/CircularProgress';
import Tooltip from '@mui/material/Tooltip';
import PushPinIcon from '@mui/icons-material/PushPin';
import { useQuery } from '@tanstack/react-query';
import { apiInstance } from '../../services/api';
import type { Feedback } from '../feedback/Feedback';
import { classifiedFeedbacks } from '../analyse/shared';
import { POSTIT_COLORS } from './data';

type ApiPostit = {
  id: number;
  message_id: string;
  texte: string;
  cree_le: string;
  auteur_nom: string;
  auteur_prenom: string;
  feedback_content: string;
  categorie: string | null;
  resume: string | null;
  sentiment: string | null;
  urgence: number | null;
};

const BOARD_WIDTH = 900;
const BOARD_HEIGHT = 660;
const POSTIT_W = 180;

function postitPosition(idx: number, id: number) {
  const col = idx % 4;
  const row = Math.floor(idx / 4);
  return {
    x: 30 + col * 215 + (id % 3) * 12,
    y: 55 + row * 195 + (id % 5) * 9,
    rotation: ((id * 7) % 70) / 10 - 3.5,
  };
}

interface Props {
  feedbacks: Feedback[];
  onPinNote: () => void;
}

export default function RightPanelPublic({ feedbacks, onPinNote }: Props) {
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [hovered, setHovered] = useState<number | null>(null);

  const { data: postits = [], isLoading } = useQuery<ApiPostit[]>({
    queryKey: ['postits'],
    queryFn: async () => {
      const res = await apiInstance.get<ApiPostit[]>('/api/v2/postit');
      return res.data ?? [];
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const categories = useMemo(() => {
    const fromFeedbacks = classifiedFeedbacks(feedbacks).map(f => f.categorie).filter(Boolean) as string[];
    const fromPostits = postits.map(p => p.categorie).filter(Boolean) as string[];
    return Array.from(new Set([...fromFeedbacks, ...fromPostits])).sort();
  }, [feedbacks, postits]);

  const visible = postits.filter(p =>
    activeCategory === null || (p.categorie ?? '').toLowerCase() === activeCategory.toLowerCase()
  );

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
        {isLoading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 200 }}>
            <CircularProgress size={28} />
          </Box>
        ) : (
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
            {visible.length === 0 && (
              <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Typography color="text.disabled" variant="body2">Aucune note épinglée pour le moment.</Typography>
              </Box>
            )}

            {visible.map((postit, idx) => {
              const { x, y, rotation } = postitPosition(idx, postit.id);
              const bgColor = POSTIT_COLORS[postit.id % POSTIT_COLORS.length].value;
              const auteur = `${postit.auteur_nom} ${postit.auteur_prenom}`.toUpperCase();
              const timestamp = new Date(postit.cree_le).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
              const tag = postit.categorie || 'Note';

              return (
                <Tooltip
                  key={postit.id}
                  title={
                    <Box sx={{ maxWidth: 280 }}>
                      <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }}>Message original :</Typography>
                      <Typography variant="caption" sx={{ display: 'block', mb: postit.resume ? 1 : 0 }}>{postit.feedback_content}</Typography>
                      {postit.resume && (
                        <>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }}>Résumé IA :</Typography>
                          <Typography variant="caption" sx={{ display: 'block' }}>{postit.resume}</Typography>
                        </>
                      )}
                    </Box>
                  }
                  placement="top"
                  arrow
                >
                  <Box
                    onMouseEnter={() => setHovered(postit.id)}
                    onMouseLeave={() => setHovered(null)}
                    sx={{
                      position: 'absolute',
                      left: x,
                      top: y,
                      width: POSTIT_W,
                      transform: `rotate(${rotation}deg) scale(${hovered === postit.id ? 1.06 : 1})`,
                      transition: 'transform 0.2s ease, box-shadow 0.2s ease',
                      zIndex: hovered === postit.id ? 20 : 1,
                      cursor: 'default',
                    }}
                  >
                    <Paper
                      elevation={hovered === postit.id ? 8 : 2}
                      sx={{ bgcolor: bgColor, borderRadius: '2px', overflow: 'visible' }}
                    >
                      {/* Pin */}
                      <Box sx={{
                        position: 'absolute', top: -9, left: '50%', transform: 'translateX(-50%)',
                        width: 18, height: 18, borderRadius: '50%',
                        background: 'radial-gradient(circle at 38% 35%, #f87171, #dc2626)',
                        boxShadow: '0 2px 4px rgba(0,0,0,0.25)', zIndex: 2,
                      }} />

                      <Box sx={{ p: 1.5, pt: 1.75 }}>
                        {/* Author */}
                        <Typography variant="caption" sx={{
                          display: 'block', fontWeight: 700,
                          letterSpacing: '0.05em', color: '#6b7280',
                          fontSize: '0.6rem', mb: 0.75,
                        }}>
                          {auteur}
                        </Typography>

                        {/* Note text */}
                        <Typography variant="body2" sx={{
                          fontSize: '0.78rem', lineHeight: 1.45, color: '#1f2937', mb: 1.25,
                          display: '-webkit-box',
                          WebkitLineClamp: 4,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}>
                          {postit.texte}
                        </Typography>

                        {/* Footer */}
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Chip
                            label={tag}
                            size="small"
                            sx={{ height: 17, fontSize: '0.6rem', fontWeight: 700, bgcolor: 'rgba(0,0,0,0.08)', color: '#374151' }}
                          />
                          <Typography variant="caption" sx={{ fontSize: '0.62rem', color: '#9ca3af' }}>
                            {timestamp}
                          </Typography>
                        </Box>

                        {/* Urgence badge */}
                        {postit.urgence != null && postit.urgence >= 3 && (
                          <Box sx={{ mt: 0.75 }}>
                            <Chip
                              label={`Urgence ${postit.urgence}`}
                              size="small"
                              sx={{ height: 15, fontSize: '0.58rem', fontWeight: 700, bgcolor: '#fee2e2', color: '#dc2626' }}
                            />
                          </Box>
                        )}
                      </Box>
                    </Paper>
                  </Box>
                </Tooltip>
              );
            })}
          </Box>
        )}
      </Box>
    </Box>
  );
}
