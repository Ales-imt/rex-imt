import { useState, useEffect } from 'react';
import Box from '@mui/material/Box';
import Drawer from '@mui/material/Drawer';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';
import PushPinIcon from '@mui/icons-material/PushPin';
import CloseIcon from '@mui/icons-material/Close';
import type { Feedback } from '../feedback/Feedback';
import { colorFromUrgence } from './data';

interface PinDrawerProps {
  open: boolean;
  onClose: () => void;
  onPin: (reponse: string, messageModere: string) => void;
  feedback: Feedback | null;
}

const MAX_REPONSE = 160;

export default function PinDrawer({ open, onClose, onPin, feedback }: PinDrawerProps) {
  const [reponse, setReponse] = useState('');
  const [messageModere, setMessageModere] = useState('');

  useEffect(() => {
    if (open && feedback) {
      setMessageModere(feedback.content);
      setReponse('');
    }
  }, [open, feedback]);

  const noteColor = colorFromUrgence(feedback?.urgence);
  const authorLabel = feedback?.groupe || feedback?.promotion || 'Étudiant';

  const handlePin = () => {
    onPin(reponse, messageModere);
    onClose();
  };

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { borderRadius: '16px 16px 0 0', maxHeight: '80vh', overflow: 'visible' } }}
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

      {/* Content */}
      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 210px', gap: 3, px: 3, pb: 3, overflow: 'auto' }}>

        {/* Left column */}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>

          {/* Student badge */}
          <Chip
            label={authorLabel}
            size="small"
            sx={{ alignSelf: 'flex-start', bgcolor: '#f0fdf4', color: '#166534', fontWeight: 700, border: '1px solid #86efac' }}
          />

          {/* Full message — read-only, for moderation */}
          <Box>
            <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 0.75 }}>
              Message complet
            </Typography>
            <Box sx={{
              maxHeight: 160, overflowY: 'auto',
              border: '1px solid', borderColor: 'divider',
              borderRadius: 1, px: 1.5, py: 1,
            }}>
              <Typography variant="body2" sx={{ fontStyle: 'italic', color: 'text.secondary', whiteSpace: 'pre-wrap' }}>
                {feedback?.content}
              </Typography>
            </Box>
          </Box>

          {/* message_modere — editable public version */}
          <Box>
            <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 0.75 }}>
              Message modéré <Typography component="span" variant="caption" color="text.disabled">(version affichée sur le mur)</Typography>
            </Typography>
            <TextField
              multiline minRows={2} maxRows={4} fullWidth
              placeholder="Version publique du message…"
              value={messageModere}
              onChange={e => setMessageModere(e.target.value)}
              size="small"
            />
          </Box>

          {/* reponse — moderator's reply */}
          <Box>
            <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 0.75 }}>
              Réponse du modérateur
            </Typography>
            <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
              <TextField
                multiline minRows={2} maxRows={4} fullWidth
                placeholder="Votre réponse affichée sur la note…"
                value={reponse}
                onChange={e => { if (e.target.value.length <= MAX_REPONSE) setReponse(e.target.value); }}
                variant="standard"
                inputProps={{ maxLength: MAX_REPONSE }}
                sx={{
                  px: 1.5, py: 1,
                  '& .MuiInput-root': { fontSize: '0.875rem' },
                  '& .MuiInput-root::before, & .MuiInput-root::after': { display: 'none' },
                }}
              />
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', px: 1.5, pb: 0.75 }}>
                <Typography variant="caption" color={reponse.length >= MAX_REPONSE ? 'error' : 'text.disabled'}>
                  {reponse.length}/{MAX_REPONSE}
                </Typography>
              </Box>
            </Paper>
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
              disabled={!reponse.trim()}
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

        {/* Right column — preview */}
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
              bgcolor: noteColor, borderRadius: '2px',
              p: 1.5, pt: 1.75, transform: 'rotate(-1deg)', minHeight: 160,
            }}>
              <Typography variant="caption" sx={{
                display: 'block', fontWeight: 700,
                letterSpacing: '0.05em', color: '#6b7280',
                fontSize: '0.6rem', mb: 0.75,
              }}>
                {authorLabel.toUpperCase()}
              </Typography>

              {messageModere && (
                <Typography variant="body2" sx={{
                  fontSize: '0.7rem', lineHeight: 1.35, color: '#374151',
                  mb: 0.75, fontStyle: 'italic',
                  display: '-webkit-box', WebkitLineClamp: 3,
                  WebkitBoxOrient: 'vertical', overflow: 'hidden',
                }}>
                  {messageModere}
                </Typography>
              )}

              <Typography variant="body2" sx={{
                fontSize: '0.78rem', lineHeight: 1.45, color: '#1f2937', mb: 1,
                display: '-webkit-box', WebkitLineClamp: 3,
                WebkitBoxOrient: 'vertical', overflow: 'hidden',
              }}>
                {reponse || <span style={{ color: '#9ca3af', fontStyle: 'italic' }}>Votre réponse…</span>}
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
