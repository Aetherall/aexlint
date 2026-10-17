import type { Instrument } from "../instrument.js";

declare function archive(detail: unknown): void;

export function catalogue(instrument: Instrument): unknown {
  let detail: unknown;
  if (instrument.kind === "violin") detail = instrument.bow;
  else if (instrument.kind === "oboe") detail = instrument.reed;
  archive(detail);
  return detail;
}
