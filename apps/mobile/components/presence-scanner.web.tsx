import { useTheme } from '@/hooks/use-theme';
import { useEffect, useRef, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';

type Props = {
  active: boolean;
  onScan: (code: string) => void;
};

type Status = 'init' | 'scanning' | 'denied' | 'unavailable';

const CONTAINER_ID = 'presence-qr-reader';
const CORNER = 28;
const CORNER_THICKNESS = 3;

async function waitForElement(id: string, timeoutMs = 2000): Promise<HTMLElement | null> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const el = document.getElementById(id);
    if (el) return el;
    await new Promise(r => setTimeout(r, 50));
  }
  return null;
}

export function PresenceScanner({ active, onScan }: Props) {
  const colors = useTheme();
  const [status, setStatus] = useState<Status>('init');
  const [fallbackKey, setFallbackKey] = useState(0);
  const [videoRect, setVideoRect] = useState<{ width: number; height: number } | null>(null);
  const html5QrRef = useRef<{ stop: () => Promise<void>; clear: () => void } | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  const onScanRef = useRef(onScan);
  onScanRef.current = onScan;
  const lastScanRef = useRef(0);

  useEffect(() => {
    if (!active) return;
    let cancelled = false;

    function reportScan(text: string) {
      const now = Date.now();
      if (now - lastScanRef.current < 1500) return;
      lastScanRef.current = now;
      onScanRef.current(text);
    }

    async function start() {
      try {
        const { Html5Qrcode } = await import('html5-qrcode');
        if (cancelled) return;

        // Force un nouveau nœud DOM — le div n'existe pas encore (status=init)
        setStatus('scanning');
        setFallbackKey(prev => prev + 1);

        // Attend que React ait rendu le nouveau div
        const container = await waitForElement(CONTAINER_ID);
        if (cancelled || !container) return;

        const scanner = new Html5Qrcode(CONTAINER_ID);
        html5QrRef.current = scanner;
        await scanner.start(
          { facingMode: 'environment' },
          { fps: 10, videoConstraints: { width: { ideal: 1280 }, height: { ideal: 720 } } },
          (text: string) => reportScan(text),
          () => {}
        );
        if (cancelled) {
          scanner.stop().catch(() => {}).finally(() => scanner.clear());
          return;
        }

        // html5-qrcode fixe des px sur le div et la vidéo — on les écrase pour qu'ils suivent le container
        container.style.width = '100%';
        container.style.height = '100%';
        const video = document.querySelector<HTMLVideoElement>(`#${CONTAINER_ID} video`);
        if (video) {
          video.style.width = '100%';
          video.style.height = '100%';
          video.style.objectFit = 'cover';
          const observer = new ResizeObserver(entries => {
            const { width, height } = entries[0].contentRect;
            if (width && height) setVideoRect({ width, height });
          });
          observer.observe(video);
          resizeObserverRef.current = observer;
        }
      } catch (e) {
        console.log('[QR] erreur:', e);
        if (!cancelled) setStatus('denied');
      }
    }

    if (typeof navigator !== 'undefined' && navigator.mediaDevices) {
      start();
    } else {
      setStatus('unavailable');
    }

    return () => {
      cancelled = true;
      resizeObserverRef.current?.disconnect();
      resizeObserverRef.current = null;
      if (html5QrRef.current) {
        const scanner = html5QrRef.current;
        html5QrRef.current = null;
        scanner.stop().catch(() => {}).finally(() => scanner.clear());
      }
    };
  }, [active]);

  if (status === 'denied') {
    return (
      <View style={[styles.box, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
        <Text style={[styles.message, { color: colors.textSecondary }]}>
          Caméra indisponible (permission refusée ou non supportée). Utilisez la saisie manuelle ci-dessous.
        </Text>
      </View>
    );
  }

  if (status === 'unavailable') {
    return (
      <View style={[styles.box, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
        <Text style={[styles.message, { color: colors.textSecondary }]}>
          Scan caméra non disponible sur ce navigateur. Utilisez la saisie manuelle ci-dessous.
        </Text>
      </View>
    );
  }

  const scanBoxSize = videoRect ? Math.floor(Math.min(videoRect.width, videoRect.height) * 0.6) : 216;

  return (
    <View style={[styles.box, { borderColor: colors.cardBorder }]}>
      {/* Le div n'existe que quand status=scanning : waitForElement ne peut trouver que le bon nœud */}
      {status === 'scanning' && (
        <div key={fallbackKey} id={CONTAINER_ID} style={{ width: '100%', height: '100%' }} />
      )}
      <View style={styles.overlay} pointerEvents="none">
        <View style={{ width: scanBoxSize, height: scanBoxSize }}>
          <View style={[styles.corner, styles.cornerTL]} />
          <View style={[styles.corner, styles.cornerTR]} />
          <View style={[styles.corner, styles.cornerBL]} />
          <View style={[styles.corner, styles.cornerBR]} />
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  box: {
    borderRadius: 12,
    borderWidth: 1,
    overflow: 'hidden',
    width: '100%',
    aspectRatio: 16 / 9,
    justifyContent: 'center',
    alignItems: 'center',
  },
  overlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'center',
    alignItems: 'center',
  },
  corner: {
    position: 'absolute',
    width: CORNER,
    height: CORNER,
    borderColor: 'rgba(255,255,255,0.9)',
    borderWidth: CORNER_THICKNESS,
  },
  cornerTL: { top: 0, left: 0, borderRightWidth: 0, borderBottomWidth: 0, borderTopLeftRadius: 4 },
  cornerTR: { top: 0, right: 0, borderLeftWidth: 0, borderBottomWidth: 0, borderTopRightRadius: 4 },
  cornerBL: { bottom: 0, left: 0, borderRightWidth: 0, borderTopWidth: 0, borderBottomLeftRadius: 4 },
  cornerBR: { bottom: 0, right: 0, borderLeftWidth: 0, borderTopWidth: 0, borderBottomRightRadius: 4 },
  message: {
    fontSize: 13,
    textAlign: 'center',
    padding: 16,
  },
});
