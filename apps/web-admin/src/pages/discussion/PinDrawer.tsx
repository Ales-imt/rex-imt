import React, { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Drawer from '@mui/material/Drawer';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Paper from '@mui/material/Paper';
import PushPinIcon from '@mui/icons-material/PushPin';
import CloseIcon from '@mui/icons-material/Close';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatItalicIcon from '@mui/icons-material/FormatItalic';
import LinkIcon from '@mui/icons-material/Link';
import EmojiEmotionsIcon from '@mui/icons-material/EmojiEmotions';
import type { Feedback } from '../feedback/Feedback';
import { POSTIT_COLORS } from './data';

interface PinDrawerProps {
  open: boolean;
  onClose: () => void;
  onPin: (text: string, color: string) => void;
  feedback: Feedback | null;
}

const MAX_CHARS = 120;

export default function PinDrawer({ open, onClose, onPin, feedback }: PinDrawerProps) {
  const [text, setText] = useState('');
  const [selectedColor, setSelectedColor] = useState(POSTIT_COLORS[0].value);

  useEffect(() => {
    if (open && feedback) {
      setText(feedback.content.slice(0, MAX_CHARS));
    }
  }, [open, feedback]);

  const handlePin = () => {
    onPin(text, selectedColor);
    onClose();
  };

  const authorLabel = feedback?.groupe || feedback?.promotion || 'Étudiant';

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { borderRadius: '16px 16px 0 0', maxHeight: '70vh', overflow: 'visible' } }}
    >
      {/* Handle */}
      <Box sx={{ display: 'flex', justifyContent: 'center', pt: 1.5, pb: 0.5 }}>
        <Box sx={{ width: 36, height: 4, bgcolor: 'divider', borderRadius: '999px' }} />
      </Box>

      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 3, pb: 1.5 }}>
        <Typography variant="h6" fontWeight={700} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <PushPinIcon sx={{ color: '#f97316' }} />
          Épingler une note
        </Typography>
        <IconButton size="small" onClick={onClose}><CloseIcon /></IconButton>
      </Box>

      {/* Two-column content */}
      <Box sx={{
        display: 'grid', gridTemplateColumns: '1fr 210px', gap: 3,
        px: 3, pb: 3, overflow: 'auto',
      }}>
        {/* Editor */}
        <Box>
          {/* Student badge */}
          <Box sx={{ mb: 1.5 }}>
            <Chip
              label={authorLabel}
              size="small"
              sx={{ bgcolor: '#f0fdf4', color: '#166534', fontWeight: 700, border: '1px solid #86efac' }}
            />
          </Box>

          {/* Original message */}
          {feedback && (
            <Box sx={{
              borderLeft: '3px solid #f97316', pl: 1.5, py: 0.75,
              bgcolor: '#fff7ed', borderRadius: '0 4px 4px 0', mb: 2,
            }}>
              <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 0.25 }}>
                Message original
              </Typography>
              <Typography variant="body2" sx={{ fontStyle: 'italic', color: 'text.secondary' }}>
                {feedback.content}
              </Typography>
            </Box>
          )}

          {/* Editable textarea */}
          <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
            <Box sx={{ display: 'flex', gap: 0.25, px: 1, py: 0.5, borderBottom: '1px solid', borderColor: 'divider' }}>
              {[
                { icon: <FormatBoldIcon fontSize="small" />, label: 'Gras' },
                { icon: <FormatItalicIcon fontSize="small" />, label: 'Italique' },
                { icon: <LinkIcon fontSize="small" />, label: 'Lien' },
                { icon: <EmojiEmotionsIcon fontSize="small" />, label: 'Emoji' },
              ].map(btn => (
                <Tooltip key={btn.label} title={btn.label}>
                  <IconButton size="small" sx={{ color: 'text.secondary' }}>{btn.icon}</IconButton>
                </Tooltip>
              ))}
            </Box>
            <TextField
              multiline minRows={3} maxRows={5} fullWidth
              placeholder="Texte de la note épinglée…"
              value={text}
              onChange={e => { if (e.target.value.length <= MAX_CHARS) setText(e.target.value); }}
              variant="standard"
              inputProps={{ maxLength: MAX_CHARS }}
              sx={{
                px: 1.5, py: 1,
                '& .MuiInput-root': { fontSize: '0.875rem' },
                '& .MuiInput-root::before, & .MuiInput-root::after': { display: 'none' },
              }}
            />
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', px: 1.5, pb: 0.75 }}>
              <Typography variant="caption" color={text.length >= MAX_CHARS ? 'error' : 'text.disabled'}>
                {text.length}/{MAX_CHARS}
              </Typography>
            </Box>
          </Paper>

          {/* Color picker */}
          <Box sx={{ mt: 2, mb: 2.5 }}>
            <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 1 }}>
              Couleur de la note
            </Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              {POSTIT_COLORS.map(color => (
                <Tooltip key={color.value} title={color.label}>
                  <Box
                    onClick={() => setSelectedColor(color.value)}
                    sx={{
                      width: 28, height: 28, borderRadius: '50%', bgcolor: color.value,
                      cursor: 'pointer', border: '2px solid',
                      borderColor: selectedColor === color.value ? 'text.primary' : 'transparent',
                      boxShadow: selectedColor === color.value
                        ? '0 0 0 2px white inset'
                        : '0 1px 3px rgba(0,0,0,0.15)',
                      transition: 'transform 0.15s',
                      '&:hover': { transform: 'scale(1.15)' },
                    }}
                  />
                </Tooltip>
              ))}
            </Box>
          </Box>

          {/* Buttons */}
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            <Button variant="outlined" onClick={onClose} sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}>
              Annuler
            </Button>
            <Button
              variant="contained"
              startIcon={<PushPinIcon />}
              onClick={handlePin}
              disabled={!text.trim()}
              sx={{
                borderRadius: 2, textTransform: 'none', fontWeight: 600, boxShadow: 'none',
                background: 'linear-gradient(135deg, #f97316, #f59e0b)',
                '&:hover': { background: 'linear-gradient(135deg, #ea6c00, #e8930a)' },
              }}
            >
              Épingler
            </Button>
          </Box>
        </Box>

        {/* Live preview */}
        <Box>
          <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 1 }}>
            Aperçu
          </Typography>
          <Box sx={{ position: 'relative', width: 180 }}>
            <Box sx={{
              position: 'absolute', top: -9, left: '50%', transform: 'translateX(-50%)',
              width: 18, height: 18, borderRadius: '50%',
              background: 'radial-gradient(circle at 38% 35%, #f87171, #dc2626)',
              boxShadow: '0 2px 4px rgba(0,0,0,0.25)', zIndex: 2,
            }} />
            <Paper elevation={3} sx={{
              bgcolor: selectedColor, borderRadius: '2px',
              p: 1.5, pt: 1.75, transform: 'rotate(-1deg)', minHeight: 140,
            }}>
              <Typography variant="caption" sx={{
                display: 'block', fontWeight: 700,
                letterSpacing: '0.05em', color: '#6b7280',
                fontSize: '0.6rem', mb: 0.75,
              }}>
                {authorLabel.toUpperCase()}
              </Typography>
              <Typography variant="body2" sx={{ fontSize: '0.78rem', lineHeight: 1.45, color: '#1f2937', mb: 1 }}>
                {text || <span style={{ color: '#9ca3af', fontStyle: 'italic' }}>Votre texte ici…</span>}
              </Typography>
              <Chip
                label={feedback?.categorie || 'Note'}
                size="small"
                sx={{ height: 17, fontSize: '0.6rem', fontWeight: 700, bgcolor: 'rgba(0,0,0,0.08)', color: '#374151' }}
              />
            </Paper>
          </Box>
          <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mt: 1, fontSize: '0.65rem' }}>
            visible par toute la classe
          </Typography>
        </Box>
      </Box>
    </Drawer>
  );
}
