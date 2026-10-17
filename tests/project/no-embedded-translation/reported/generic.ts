import type { Instrument } from "../instrument.js";

export function stringsOf<T extends Instrument>(instrument: T, seen: Set<T>): number {
  let strings = 0;
  if (instrument.kind === "violin") {
    strings = instrument.strings;
    seen.add(instrument);
  }
  return strings;
}
