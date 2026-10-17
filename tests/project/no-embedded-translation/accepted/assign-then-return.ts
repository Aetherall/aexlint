import type { Instrument } from "../instrument.js";

declare class Listing {
  constructor(fields: { kind: string; detail: unknown });
}

export function listingOf(instrument: Instrument): Listing {
  let detail: unknown;
  if (instrument.kind === "violin") detail = instrument.bow;
  else if (instrument.kind === "oboe") detail = instrument.reed;
  return new Listing({ kind: instrument.kind, detail });
}

export function labelOf(instrument: Instrument): unknown {
  let label: unknown = "percussion";
  switch (instrument.kind) {
    case "violin":
      label = instrument.strings;
      break;
    case "oboe":
      label = instrument.keys;
      break;
  }
  return label;
}
