import type { Instrument } from "../instrument.js";

export function keysOf(instrument: Instrument, counts: number[]): void {
  const { kind } = instrument;
  let keys = 0;
  if (kind === "oboe") keys = instrument.keys;
  counts.push(keys);
}
