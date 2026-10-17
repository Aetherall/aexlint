import type { Instrument } from "../instrument.js";

export interface Sound {
  sources: unknown[];
}

export function soundOf(instrument: Instrument): Sound | undefined {
  switch (instrument.kind) {
    case "violin":
      return { sources: [instrument.bow, instrument.strings] };
    case "oboe":
      return { sources: [instrument.reed] };
    default:
      return undefined;
  }
}

declare function loudness(source: unknown): number;
const levels = new Map<Instrument, number>();

export function measureSound(instrument: Instrument): void {
  const sound = soundOf(instrument);
  levels.set(instrument, sound ? Math.max(...sound.sources.map(loudness)) : 0);
}
