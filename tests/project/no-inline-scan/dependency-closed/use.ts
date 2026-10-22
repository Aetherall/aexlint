import { Sink } from "./sink.js";

export function offsets(widths: number[], sink: Sink): Sink {
  let offset = 0;
  for (const width of widths) {
    sink.put(offset);
    offset += width;
  }
  return sink;
}
