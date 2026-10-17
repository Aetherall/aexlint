import type { Instrument } from "../instrument.js";

declare function loudness(source: unknown): number;
const levels = new Map<Instrument, number>();

export function measure(instrument: Instrument): void {
  let sources: unknown[];
  switch (instrument.kind) {
    case "violin":
      sources = [instrument.bow, instrument.strings];
      break;
    case "oboe":
      sources = [instrument.reed];
      break;
    default:
      levels.set(instrument, 0);
      return;
  }
  levels.set(instrument, Math.max(...sources.map(loudness)));
}
