import type { Instrument } from "../instrument.js";

export function describe(instrument: Instrument): string {
  if (instrument.kind === "violin") {
    instrument.bow;
    return `${instrument.strings} strings`;
  }
  return instrument.kind;
}
