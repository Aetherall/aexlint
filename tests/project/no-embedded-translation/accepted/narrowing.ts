import type { Instrument } from "../instrument.js";

export function reedOf(instrument: Instrument) {
  if (instrument.kind === "oboe") return instrument.reed;
  return undefined;
}

export const pedalsOf = (instrument: Instrument) =>
  instrument.kind === "timpani" ? instrument.pedals : 0;
