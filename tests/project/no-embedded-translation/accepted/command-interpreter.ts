import type { Instrument } from "../instrument.js";

declare function restring(count: number): Promise<void>;
declare function soak(reed: unknown): Promise<void>;

export async function service(instrument: Instrument): Promise<string | undefined> {
  switch (instrument.kind) {
    case "violin":
      await restring(instrument.strings);
      return "restrung";
    case "oboe":
      await soak(instrument.reed);
      return "soaked";
    default:
      return undefined;
  }
}
