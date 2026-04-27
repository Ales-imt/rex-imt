import { useState, useEffect, useMemo, useRef } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Paper from '@mui/material/Paper';
import TextField from '@mui/material/TextField';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Avatar from '@mui/material/Avatar';
import CircularProgress from '@mui/material/CircularProgress';
import SendIcon from '@mui/icons-material/Send';
import FormatBoldIcon from '@mui/icons-material/FormatBold';
import FormatItalicIcon from '@mui/icons-material/FormatItalic';
import LinkIcon from '@mui/icons-material/Link';
import type { Feedback } from '../feedback/Feedback';
import { urgenceColor, relativeTime, SENTIMENT_LABEL, SENTIMENT_COLOR } from '../analyse/shared';
import { apiInstance } from '../../services/api';

type Reponse = {
  id: number;
  message_id: string;
  contenu: string;
  auteur: string;
  cree_le: string;
  lu: boolean;
};

type ThreadItem =
  | { kind: 'feedback'; fb: Feedback; time: number }
  | { kind: 'reponse'; rep: Reponse; time: number };

interface RightPanelPriveProps {
  feedback: Feedback | null;
  allFeedbacks: Feedback[];
}

export default function RightPanelPrive({ feedback, allFeedbacks }: RightPanelPriveProps) {
  const [draft, setDraft] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [threadData, setThreadData] = useState<Map<number, Reponse[]>>(new Map());
  const [loadingThread, setLoadingThread] = useState(false);
  const selectedRef = useRef<HTMLDivElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);

  // All feedbacks from the same pseudo, sorted chronologically
  const pseudoFeedbacks = useMemo(() => {
    if (!feedback) return [];
    if (!feedback.pseudo) return [feedback];
    return [...allFeedbacks]
      .filter(fb => fb.pseudo === feedback.pseudo)
      .sort((a, b) => new Date(a.created_at ?? 0).getTime() - new Date(b.created_at ?? 0).getTime());
  }, [feedback?.pseudo, feedback?.id, allFeedbacks]);

  const pseudoIdsKey = pseudoFeedbacks.map(f => f.id).join(',');

  // Fetch responses for every feedback of this pseudo
  useEffect(() => {
    if (pseudoFeedbacks.length === 0) {
      setThreadData(new Map());
      return;
    }
    setLoadingThread(true);
    let cancelled = false;

    Promise.all(
      pseudoFeedbacks.map(fb =>
        apiInstance.get<Reponse[]>(`/api/v2/reponse/${fb.id}`)
          .then(res => ({ id: fb.id, reps: res.data ?? [] }))
          .catch(() => ({ id: fb.id, reps: [] }))
      )
    ).then(results => {
      if (cancelled) return;
      const map = new Map<number, Reponse[]>();
      results.forEach(({ id, reps }) => map.set(id, reps));
      setThreadData(map);
      setLoadingThread(false);
    });

    return () => { cancelled = true; };
  }, [pseudoIdsKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Merge feedbacks + responses into a single chronological list
  const threadItems = useMemo((): ThreadItem[] => {
    const items: ThreadItem[] = [
      ...pseudoFeedbacks.map(fb => ({
        kind: 'feedback' as const,
        fb,
        time: new Date(fb.created_at ?? 0).getTime(),
      })),
      ...Array.from(threadData.values()).flat().map(rep => ({
        kind: 'reponse' as const,
        rep,
        time: new Date(rep.cree_le).getTime(),
      })),
    ];
    return items.sort((a, b) => a.time - b.time);
  }, [pseudoFeedbacks, threadData]);

  // Scroll to selected feedback when selection changes
  useEffect(() => {
    selectedRef.current?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }, [feedback?.id]);

  if (!feedback) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <Typography color="text.secondary">Sélectionnez un feedback dans la liste.</Typography>
      </Box>
    );
  }

  const color = urgenceColor(feedback.urgence);
  const sentimentColor = feedback.sentiment ? SENTIMENT_COLOR[feedback.sentiment] : undefined;

  const handleSend = async () => {
    const contenu = draft.trim();
    if (!contenu) return;
    setSending(true);
    setError('');
    try {
      const res = await apiInstance.post<Reponse>('/api/v2/reponse', {
        feedback_id: feedback.id,
        contenu,
      });
      setThreadData(prev => {
        const next = new Map(prev);
        next.set(feedback.id, [...(next.get(feedback.id) ?? []), res.data]);
        return next;
      });
      setDraft('');
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50);
    } catch {
      setError("Échec de l'envoi. Réessayez.");
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSend();
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: 'background.default' }}>
      {/* Header */}
      <Box sx={{
        px: 3, py: 1.5,
        borderBottom: '1px solid', borderColor: 'divider',
        bgcolor: 'background.paper',
        display: 'flex', alignItems: 'center', gap: 1.5,
      }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="subtitle1" fontWeight={700} noWrap>
            {feedback.resume || feedback.categorie || 'Feedback étudiant'}
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {feedback.promotion && (
              <Typography variant="caption" color="text.secondary">{feedback.promotion}</Typography>
            )}
            {pseudoFeedbacks.length > 1 && (
              <Chip label={`${pseudoFeedbacks.length} messages`} size="small" variant="outlined" sx={{ height: 18, fontSize: '0.65rem' }} />
            )}
          </Box>
        </Box>
        <Box sx={{ display: 'flex', gap: 0.75, flexShrink: 0, flexWrap: 'wrap', justifyContent: 'flex-end' }}>
          {feedback.urgence != null && (
            <Chip
              label={`Urgence ${feedback.urgence}`}
              size="small"
              sx={{ bgcolor: color, color: feedback.urgence <= 1 ? '#555' : '#fff', fontWeight: 700, fontSize: '0.7rem' }}
            />
          )}
          {feedback.sentiment && sentimentColor && (
            <Chip
              label={SENTIMENT_LABEL[feedback.sentiment] ?? feedback.sentiment}
              size="small"
              sx={{ bgcolor: sentimentColor + '22', color: sentimentColor, fontWeight: 600, fontSize: '0.7rem' }}
            />
          )}
          {feedback.resume && (
            <Chip
              label="Résumé IA"
              size="small"
              sx={{ background: 'linear-gradient(135deg, #667eea, #764ba2)', color: '#fff', fontWeight: 700, fontSize: '0.7rem' }}
            />
          )}
        </Box>
      </Box>

      {/* Thread */}
      <Box sx={{ flex: 1, overflowY: 'auto', px: 3, py: 2 }}>
        {/* AI summary */}
        {feedback.resume && (
          <Box sx={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', borderRadius: 2, p: '2px', mb: 2.5 }}>
            <Box sx={{ bgcolor: 'background.paper', borderRadius: '6px', p: 1.75, display: 'flex', alignItems: 'flex-start', gap: 1.5 }}>
              <Box sx={{
                width: 36, height: 36, borderRadius: '50%',
                background: 'linear-gradient(135deg, #667eea, #764ba2)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: '1rem', flexShrink: 0,
              }}>🤖</Box>
              <Box>
                <Typography variant="caption" fontWeight={700} sx={{ color: '#7c3aed', display: 'block', mb: 0.25 }}>
                  Résumé IA · Analyse thématique
                </Typography>
                <Typography variant="body2" color="text.secondary">{feedback.resume}</Typography>
              </Box>
            </Box>
          </Box>
        )}

        {loadingThread && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
            <CircularProgress size={20} />
          </Box>
        )}

        {/* Chronological thread: feedbacks left, responses right */}
        {threadItems.map((item, idx) => {
          const prev = threadItems[idx - 1];
          const showDateSep = idx === 0 ||
            new Date(item.time).toDateString() !== new Date(prev.time).toDateString();

          const dateSep = showDateSep && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, my: 2 }}>
              <Box sx={{ flex: 1, height: 1, bgcolor: 'divider' }} />
              <Typography variant="caption" color="text.disabled">
                {new Date(item.time).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })}
              </Typography>
              <Box sx={{ flex: 1, height: 1, bgcolor: 'divider' }} />
            </Box>
          );

          if (item.kind === 'feedback') {
            const { fb } = item;
            const isSelected = fb.id === feedback.id;
            const fbColor = urgenceColor(fb.urgence);
            return (
              <Box key={`fb-${fb.id}`} ref={isSelected ? selectedRef : undefined}>
                {dateSep}
                {/* Feedback — LEFT */}
                <Box sx={{ display: 'flex', gap: 1, maxWidth: '72%', mb: 1.5 }}>
                  <Avatar sx={{ width: 30, height: 30, fontSize: '0.65rem', flexShrink: 0, bgcolor: fbColor }}>É</Avatar>
                  <Box sx={{ flex: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 0.25, ml: 0.5 }}>
                      <Typography variant="caption" color="text.disabled">
                        {fb.groupe || fb.promotion || 'Étudiant'} · {relativeTime(fb.created_at)}
                      </Typography>
                      {isSelected && (
                        <Chip label="sélectionné" size="small" color="primary" sx={{ height: 16, fontSize: '0.6rem', fontWeight: 700 }} />
                      )}
                    </Box>
                    <Box sx={{
                      bgcolor: isSelected ? 'primary.100' : 'action.hover',
                      border: isSelected ? '2px solid' : 'none',
                      borderColor: 'primary.main',
                      borderRadius: '4px 14px 14px 14px',
                      px: 1.75, py: 1,
                    }}>
                      <Typography variant="body2">{fb.content}</Typography>
                    </Box>
                    {(fb.categorie || fb.sous_categorie) && (
                      <Box sx={{ display: 'flex', gap: 0.5, mt: 0.5, flexWrap: 'wrap', ml: 0.5 }}>
                        {fb.categorie && <Chip label={fb.categorie} size="small" variant="outlined" sx={{ height: 18, fontSize: '0.65rem' }} />}
                        {fb.sous_categorie && <Chip label={fb.sous_categorie} size="small" variant="outlined" sx={{ height: 18, fontSize: '0.65rem' }} />}
                      </Box>
                    )}
                  </Box>
                </Box>
              </Box>
            );
          }

          // Réponse — RIGHT
          const { rep } = item;
          return (
            <Box key={`rep-${rep.id}`}>
              {dateSep}
              <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1.5 }}>
                <Box sx={{ maxWidth: '72%' }}>
                  <Typography variant="caption" color="text.disabled" sx={{ mr: 0.5, mb: 0.25, display: 'block', textAlign: 'right' }}>
                    {rep.auteur} · {relativeTime(rep.cree_le)}
                  </Typography>
                  <Box sx={{ bgcolor: '#2f5ff5', borderRadius: '14px 4px 14px 14px', px: 1.75, py: 1 }}>
                    <Typography variant="body2" sx={{ color: '#fff' }}>{rep.contenu}</Typography>
                  </Box>
                </Box>
              </Box>
            </Box>
          );
        })}

        {/* Sentinel — scroll target after send */}
        <div ref={bottomRef} />
      </Box>

      {/* Compose */}
      <Paper elevation={0} sx={{ borderTop: '1px solid', borderColor: 'divider', bgcolor: 'background.paper' }}>
        <Box sx={{ display: 'flex', gap: 0.25, px: 1.5, pt: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
          {[
            { icon: <FormatBoldIcon fontSize="small" />, label: 'Gras' },
            { icon: <FormatItalicIcon fontSize="small" />, label: 'Italique' },
            { icon: <LinkIcon fontSize="small" />, label: 'Lien' },
          ].map(btn => (
            <Tooltip key={btn.label} title={btn.label}>
              <IconButton size="small" sx={{ color: 'text.secondary', '&:hover': { color: 'text.primary' } }}>
                {btn.icon}
              </IconButton>
            </Tooltip>
          ))}
          <Typography variant="caption" color="text.disabled" sx={{ ml: 'auto', alignSelf: 'center', pr: 0.5 }}>
            Réponse au feedback sélectionné
          </Typography>
        </Box>
        {error && (
          <Typography variant="caption" color="error" sx={{ px: 2, pt: 0.75, display: 'block' }}>{error}</Typography>
        )}
        <Box sx={{ display: 'flex', alignItems: 'flex-end', gap: 1, p: 1.5 }}>
          <TextField
            multiline minRows={2} maxRows={4} fullWidth
            placeholder="Répondre à cet étudiant… (Ctrl+Entrée pour envoyer)"
            value={draft}
            onChange={e => setDraft(e.target.value)}
            onKeyDown={handleKeyDown}
            variant="outlined" size="small"
            sx={{ '& .MuiOutlinedInput-root': { borderRadius: 2, fontSize: '0.875rem' } }}
          />
          <IconButton
            onClick={handleSend}
            disabled={!draft.trim() || sending}
            sx={{
              bgcolor: '#2f5ff5', color: '#fff', borderRadius: 1.5, mb: 0.25,
              '&:hover': { bgcolor: '#1e4ed8' },
              '&.Mui-disabled': { bgcolor: 'action.disabledBackground' },
            }}
          >
            {sending ? <CircularProgress size={16} sx={{ color: '#fff' }} /> : <SendIcon fontSize="small" />}
          </IconButton>
        </Box>
      </Paper>
    </Box>
  );
}
