import type { Instrument } from "../instrument.js";

declare function tensionOf(heads: unknown): number;

export function tune(instrument: Instrument, pedals: number[]): number {
  let highest = 0;
  if (instrument.kind === "timpani") {
    highest = tensionOf(instrument.heads);
    pedals.push(instrument.pedals);
  }
  return highest;
}
