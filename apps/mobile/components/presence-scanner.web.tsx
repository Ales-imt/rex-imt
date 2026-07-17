import { useTheme } from '@/hooks/use-theme';
import { useCallback, useEffect, useRef, useState } from 'react';
import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';

type Props = {
  active: boolean;
  onScan: (code: string) => void;
};

type Status = 'init' | 'picking' | 'scanning' | 'denied' | 'unavailable';

const CONTAINER_ID = 'presence-qr-reader';
const CORNER = 28;
const CORNER_THICKNESS = 3;

// Détection des navigateurs sous Android (tous navigateurs)
const isAndroidBrowser = (): boolean => {
  const userAgent = navigator.userAgent || navigator.vendor || (window as any).opera;
  return /android/i.test(userAgent);
};

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
  const [cameras, setCameras] = useState<MediaDeviceInfo[]>([]);
  const [fallbackKey, setFallbackKey] = useState(0);
  const [videoRect, setVideoRect] = useState<{ width: number; height: number } | null>(null);
  const html5QrRef = useRef<{ stop: () => Promise<void>; clear: () => void } | null>(null);
  const jsQRAnimationFrameRef = useRef<number | null>(null);
  const jsQRCanvasRef = useRef<HTMLCanvasElement | null>(null);
  const jsQRVideoRef = useRef<HTMLVideoElement | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  const onScanRef = useRef(onScan);
  onScanRef.current = onScan;
  const lastScanRef = useRef(0);

  function reportScan(text: string) {
    const now = Date.now();
    if (now - lastScanRef.current < 1500) return;
    lastScanRef.current = now;
    onScanRef.current(text);
  }

  function stopScanner() {
    resizeObserverRef.current?.disconnect();
    resizeObserverRef.current = null;
    
    // Arrêter le scanner html5-qrcode
    if (html5QrRef.current) {
      const scanner = html5QrRef.current;
      html5QrRef.current = null;
      scanner.stop().catch(() => {}).finally(() => scanner.clear());
    }
    
    // Arrêter le scanner jsQR (Android)
    if (jsQRAnimationFrameRef.current !== null) {
      cancelAnimationFrame(jsQRAnimationFrameRef.current);
      jsQRAnimationFrameRef.current = null;
    }
    
    // Nettoyer les éléments vidéo/canvas de jsQR
    if (jsQRVideoRef.current) {
      const stream = jsQRVideoRef.current.srcObject as MediaStream | null;
      if (stream) {
        stream.getTracks().forEach(track => track.stop());
      }
      jsQRVideoRef.current.srcObject = null;
      jsQRVideoRef.current = null;
    }
    jsQRCanvasRef.current = null;
  }

  // Phase 1 : permission + énumération des caméras
  useEffect(() => {
    if (!active) {
      stopScanner();
      setStatus('init');
      setCameras([]);
      return;
    }

    let cancelled = false;

    async function init() {
      try {
        const permStream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: 'environment' } },
        });
        permStream.getTracks().forEach(t => t.stop());
        if (cancelled) return;

        const allDevices = await navigator.mediaDevices.enumerateDevices();
        const videoDevices = allDevices.filter(d => d.kind === 'videoinput') as MediaDeviceInfo[];
        if (cancelled) return;

        // Filtrer pour ne garder que les caméras arrière (environment)
        const backCameras = videoDevices.filter(cam => {
          const label = cam.label?.toLowerCase() || '';
          return label.includes('back') || label.includes('rear') || label.includes('environment') || label.includes('arrière');
        });

        // Si pas de caméra arrière trouvée, proposer toutes les caméras
        const camerasToUse = backCameras.length > 0 ? backCameras : videoDevices;
        setCameras(camerasToUse);

        // Démarrage automatique sur la première caméra de la liste ;
        // le choix reste accessible via « Changer de caméra »
        const defaultCam = camerasToUse.find(cam => cam.deviceId);
        if (defaultCam) {
          startWithCamera(defaultCam.deviceId);
        } else {
          // Aucune caméra disponible
          setStatus('unavailable');
        }
      } catch {
        if (!cancelled) setStatus('denied');
      }
    }

    if (typeof navigator !== 'undefined' && navigator.mediaDevices) {
      init();
    } else {
      setStatus('unavailable');
    }

    return () => {
      cancelled = true;
      stopScanner();
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  // Phase 2 : démarrage du scanner avec la caméra choisie
  const startWithCamera = useCallback(async (deviceId: string) => {
    setStatus('scanning');
    setFallbackKey(prev => prev + 1);

    const container = await waitForElement(CONTAINER_ID);
    if (!container) return;

    // Vider le container pour jsQR
    container.innerHTML = '';

    try {
      // --- CAS SPECIFIQUE : Android (utilise jsQR pour tous les navigateurs Android) ---
      if (isAndroidBrowser()) {
        const { default: jsQR } = await import('jsqr');
        
        // Créer l'élément vidéo
        const video = document.createElement('video');
        video.playsInline = true;
        video.style.width = '100%';
        video.style.height = '100%';
        video.style.objectFit = 'cover';
        jsQRVideoRef.current = video;
        container.appendChild(video);

        // Créer le canvas pour la détection
        const canvas = document.createElement('canvas');
        jsQRCanvasRef.current = canvas;

        // Démarrer le stream vidéo
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { deviceId: { exact: deviceId }, facingMode: 'environment' }
        });
        video.srcObject = stream;
        await video.play();

        // Fonction de scan avec jsQR
        const scanWithJsQR = () => {
          if (!video.videoWidth || !video.videoHeight) {
            jsQRAnimationFrameRef.current = requestAnimationFrame(scanWithJsQR);
            return;
          }

          // Redimensionner le canvas
          canvas.width = video.videoWidth;
          canvas.height = video.videoHeight;
          const ctx = canvas.getContext('2d');
          if (!ctx) return;
          
          ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
          const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
          
          const code = jsQR(imageData.data, imageData.width, imageData.height);
          if (code) {
            reportScan(code.data);
          }
          
          jsQRAnimationFrameRef.current = requestAnimationFrame(scanWithJsQR);
        };

        // Démarrer la boucle de scan
        scanWithJsQR();

        // Observer pour videoRect
        const observer = new ResizeObserver(entries => {
          const { width, height } = entries[0].contentRect;
          if (width && height) setVideoRect({ width, height });
        });
        observer.observe(video);
        resizeObserverRef.current = observer;

      // --- CAS GENERAL : Autres navigateurs (utilise html5-qrcode) ---
      } else {
        const { Html5Qrcode } = await import('html5-qrcode');
        const scanner = new Html5Qrcode(CONTAINER_ID);
        html5QrRef.current = scanner;

        await scanner.start(
          deviceId,
          { 
            fps: 10,
            videoConstraints: { facingMode: 'environment' }
          },
          (text: string) => reportScan(text),
          () => {}
        );

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
      }
    } catch (e) {
      console.log('[QR] erreur:', e);
      setStatus('denied');
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
    <>
      <View style={[styles.box, { borderColor: colors.cardBorder }]}>
        {status === 'init' && (
          <Text style={[styles.message, { color: colors.textSecondary }]}>Initialisation...</Text>
        )}
        {status === 'scanning' && (
          <div key={fallbackKey} id={CONTAINER_ID} style={{ width: '100%', height: '100%' }} />
        )}
        <View style={styles.overlay} pointerEvents="none">
          {status === 'scanning' && (
            <View style={{ width: scanBoxSize, height: scanBoxSize }}>
              <View style={[styles.corner, styles.cornerTL]} />
              <View style={[styles.corner, styles.cornerTR]} />
              <View style={[styles.corner, styles.cornerBL]} />
              <View style={[styles.corner, styles.cornerBR]} />
            </View>
          )}
        </View>
      </View>

      {status === 'scanning' && cameras.length > 1 && (
        <TouchableOpacity
          style={[styles.switchBtn, { borderColor: colors.cardBorder, backgroundColor: colors.cardBg }]}
          onPress={() => { stopScanner(); setStatus('picking'); }}
        >
          <Text style={[styles.switchBtnText, { color: colors.tint }]}>Changer de caméra</Text>
        </TouchableOpacity>
      )}

      {status === 'picking' && (
        <View style={styles.modalOverlay}>
          <View style={[styles.modal, { backgroundColor: colors.cardBg, borderColor: colors.cardBorder }]}>
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Choisir une caméra</Text>
            {cameras.map((cam, i) => (
              <TouchableOpacity
                key={cam.deviceId}
                style={[styles.camOption, { borderColor: colors.cardBorder }]}
                onPress={() => startWithCamera(cam.deviceId)}
              >
                <Text style={[styles.camLabel, { color: colors.textPrimary }]}>
                  {cam.label || `Caméra ${i + 1}`}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>
      )}
    </>
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
  switchBtn: {
    alignSelf: 'flex-end',
    marginTop: 6,
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 8,
    borderWidth: 1,
  },
  switchBtnText: {
    fontSize: 13,
    fontWeight: '500',
  },
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0,0,0,0.6)',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 9999,
    padding: 24,
  },
  modal: {
    width: '100%',
    maxWidth: 400,
    borderRadius: 14,
    borderWidth: 1,
    padding: 20,
    gap: 10,
  },
  modalTitle: {
    fontSize: 16,
    fontWeight: '700',
    marginBottom: 4,
  },
  camOption: {
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderRadius: 10,
    borderWidth: 1,
  },
  camLabel: {
    fontSize: 14,
  },
});
