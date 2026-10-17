import type { Instrument, Oboe, Violin } from "../instrument.js";

declare function playViolin(violin: Violin): void;
declare function playOboe(oboe: Oboe): void;

export function play(instrument: Instrument): void {
  switch (instrument.kind) {
    case "violin":
      playViolin(instrument);
      break;
    case "oboe":
      playOboe(instrument);
      break;
  }
}
