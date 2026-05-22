import { Accelerometer } from 'expo-sensors';
import { useEffect, useState } from 'react';
import { Platform } from 'react-native';

export type Orientation = 'portrait' | 'landscape-left' | 'landscape-right';

const THRESHOLD = 0.6;

function toOrientation(x: number, y: number): Orientation | null {
  if (Math.abs(x) > THRESHOLD && Math.abs(x) > Math.abs(y)) {
    return x > 0 ? 'landscape-right' : 'landscape-left';
  }
  if (Math.abs(y) > THRESHOLD && Math.abs(y) > Math.abs(x)) {
    return 'portrait';
  }
  return null; // zone ambiguë (~45°), on conserve l'état courant
}

export function useOrientation(): Orientation {
  const [orientation, setOrientation] = useState<Orientation>('portrait');

  useEffect(() => {
    if (Platform.OS === 'web') return;

    Accelerometer.setUpdateInterval(150);

    const sub = Accelerometer.addListener(({ x, y }) => {
      const next = toOrientation(x, y);
      if (next) setOrientation(next);
    });

    return () => sub.remove();
  }, []);

  return orientation;
}
