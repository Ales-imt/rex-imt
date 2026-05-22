import { useMemo, useState, useEffect, useRef } from 'react';
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
import { colorFromUrgence } from './data';

type ApiPostit = {
  id: number;
  message_id: string;
  reponse: string;
  message_modere: string | null;
  cree_le: string;
  auteur_nom: string;
  auteur_prenom: string;
  categorie: string | null;
  resume: string | null;
  sentiment: string | null;
  urgence: number | null;
};

const CORK_TEXTURE = `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='256' height='256'%3E%3Cfilter id='cork'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='4' stitchTiles='stitch'/%3E%3CfeColorMatrix type='matrix' values='0.40 0 0 0 0.30 0 0.25 0 0 0.20 0 0 0.10 0 0.08 0 0 0 1 0'/%3E%3C/filter%3E%3Crect width='256' height='256' filter='url(%23cork)'/%3E%3C/svg%3E")`;

function postitRotation(id: number) {
  return ((id * 7) % 70) / 10 - 3.5;
}

interface Props {
  feedbacks: Feedback[];
  months: number;
  promotion: string | null;
  selectedFeedbackId: number | null;
  onPinNote: () => void;
  onSelectMessage: (feedbackId: number) => void;
}

export default function RightPanelPublic({ feedbacks, months, promotion, selectedFeedbackId, onPinNote, onSelectMessage }: Props) {
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [hovered, setHovered] = useState<number | null>(null);
  const itemRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  const { data: postits = [], isLoading } = useQuery<ApiPostit[]>({
    queryKey: ['postits', months, promotion],
    queryFn: async () => {
      const params = new URLSearchParams({ months: String(months) });
      if (promotion) params.set('promotion', promotion);
      const res = await apiInstance.get<ApiPostit[]>(`/api/v2/postit?${params}`);
      return res.data ?? [];
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const feedbackByMessageId = useMemo(() => {
    const map = new Map<string, Feedback>();
    for (const f of feedbacks) {
      if (f.message_id) map.set(f.message_id, f);
    }
    return map;
  }, [feedbacks]);

  const categories = useMemo(() => {
    const fromFeedbacks = classifiedFeedbacks(feedbacks).map(f => f.categorie).filter(Boolean) as string[];
    const fromPostits = postits.map(p => p.categorie).filter(Boolean) as string[];
    return Array.from(new Set([...fromFeedbacks, ...fromPostits])).sort();
  }, [feedbacks, postits]);

  const visible = postits.filter(p =>
    activeCategory === null || (p.categorie ?? '').toLowerCase() === activeCategory.toLowerCase()
  );

  const selectedMessageId = feedbacks.find(f => f.id === selectedFeedbackId)?.message_id;

  useEffect(() => {
    if (!selectedMessageId) return;
    const highlighted = visible.find(p => p.message_id === selectedMessageId);
    if (highlighted) {
      itemRefs.current.get(highlighted.id)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }, [selectedMessageId, visible]);

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
      <Box sx={{
        flex: 1, overflowY: 'auto',
        backgroundImage: CORK_TEXTURE,
        backgroundSize: '256px 256px',
        backgroundColor: '#c8924a',
      }}>
        {isLoading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 200 }}>
            <CircularProgress size={28} />
          </Box>
        ) : visible.length === 0 ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 200 }}>
            <Typography color="rgba(255,255,255,0.7)" variant="body2">Aucune note épinglée pour le moment.</Typography>
          </Box>
        ) : (
          <Box sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(185px, 1fr))',
            gap: 3,
            p: 3,
            pt: 4,
          }}>
            {visible.map((postit) => {
              const rotation = postitRotation(postit.id);
              const urgence = feedbackByMessageId.get(postit.message_id)?.urgence ?? postit.urgence;
              const bgColor = colorFromUrgence(urgence);
              const auteur = `${postit.auteur_nom} ${postit.auteur_prenom}`.toUpperCase();
              const timestamp = new Date(postit.cree_le).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
              const tag = postit.categorie || 'Note';
              const isHighlighted = !!selectedMessageId && postit.message_id === selectedMessageId;

              return (
                <Tooltip
                  key={postit.id}
                  title={
                    <Box sx={{ maxWidth: 320, display: 'flex', flexDirection: 'column', gap: 1.25 }}>
                      {postit.message_modere && (
                        <Box>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.4 }}>Message modéré</Typography>
                          <Typography variant="caption" sx={{ display: 'block', whiteSpace: 'pre-wrap' }}>{postit.message_modere}</Typography>
                        </Box>
                      )}
                      <Box>
                        <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.4 }}>Réponse</Typography>
                        <Typography variant="caption" sx={{ display: 'block', whiteSpace: 'pre-wrap' }}>{postit.reponse}</Typography>
                      </Box>
                      {postit.resume && (
                        <Box>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.4 }}>Résumé IA</Typography>
                          <Typography variant="caption" sx={{ display: 'block', whiteSpace: 'pre-wrap' }}>{postit.resume}</Typography>
                        </Box>
                      )}
                      {postit.urgence != null && (
                        <Box>
                          <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.4 }}>Urgence</Typography>
                          <Typography variant="caption">{postit.urgence} / 5</Typography>
                        </Box>
                      )}
                    </Box>
                  }
                  placement="top"
                  arrow
                  slotProps={{
                    popper: {
                      modifiers: [
                        { name: 'flip', options: { fallbackPlacements: ['bottom', 'right', 'left'] } },
                        { name: 'preventOverflow', options: { boundary: 'viewport', padding: 8 } },
                      ],
                    },
                  }}
                >
                  <Box
                    ref={el => { if (el) itemRefs.current.set(postit.id, el as HTMLDivElement); else itemRefs.current.delete(postit.id); }}
                    onMouseEnter={() => setHovered(postit.id)}
                    onMouseLeave={() => setHovered(null)}
                    onClick={() => {
                      const fb = feedbackByMessageId.get(postit.message_id);
                      if (fb) onSelectMessage(fb.id);
                    }}
                    sx={{
                      transform: `rotate(${rotation}deg) scale(${hovered === postit.id || isHighlighted ? 1.08 : 1})`,
                      transition: 'transform 0.25s ease',
                      zIndex: isHighlighted ? 15 : hovered === postit.id ? 20 : 1,
                      position: 'relative',
                      cursor: 'default',
                    }}
                  >
                    <Paper
                      elevation={isHighlighted ? 10 : hovered === postit.id ? 8 : 2}
                      sx={{
                        bgcolor: bgColor, borderRadius: '2px', overflow: 'visible',
                        outline: isHighlighted ? '3px solid #f97316' : 'none',
                        outlineOffset: '2px',
                      }}
                    >
                      {/* Pin */}
                      <Box sx={{
                        position: 'absolute', top: -9, left: '50%', transform: 'translateX(-50%)',
                        width: 18, height: 18, borderRadius: '50%',
                        background: 'radial-gradient(circle at 38% 35%, #f87171, #dc2626)',
                        boxShadow: '0 2px 4px rgba(0,0,0,0.25)', zIndex: 2,
                      }} />

                      <Box sx={{ p: 1.5, pt: 1.75 }}>
                        <Typography variant="caption" sx={{
                          display: 'block', fontWeight: 700,
                          letterSpacing: '0.05em', color: '#6b7280',
                          fontSize: '0.6rem', mb: 0.75,
                        }}>
                          {auteur}
                        </Typography>

                        {postit.message_modere && (
                          <Typography variant="body2" sx={{
                            fontSize: '0.7rem', lineHeight: 1.35, color: '#374151',
                            fontStyle: 'italic', mb: 0.75,
                          }}>
                            {postit.message_modere.slice(0, 80)}{postit.message_modere.length > 80 ? '…' : ''}
                          </Typography>
                        )}

                        <Typography variant="body2" sx={{
                          fontSize: '0.78rem', lineHeight: 1.45, color: '#1f2937', mb: 1.25,
                        }}>
                          {postit.reponse.slice(0, 80)}{postit.reponse.length > 80 ? '…' : ''}
                        </Typography>

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
