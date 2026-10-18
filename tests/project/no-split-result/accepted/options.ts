export interface Style { width?: number; height?: number; color?: string }

export function pair(wide: boolean): Style {
  return wide ? { width: 7331 } : { height: 7332 };
}

export function sized(wide: boolean, tall: boolean): Style {
  if (wide && tall) return { width: 7333, height: 7333 };
  if (wide) return { width: 7334 };
  return { height: 7335 };
}

export function colored(wide: boolean, color: string | undefined): Style {
  if (wide) return { width: 7336, color };
  if (color) return { height: 7337, color };
  return { width: 7338, height: 7338 };
}
