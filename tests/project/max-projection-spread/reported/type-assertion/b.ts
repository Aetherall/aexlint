import type { Gauge } from "./file.js"; export const b = (g: Gauge | undefined) => ({ low: (<Gauge>g).at(7470).read(), high: (<Gauge>g).at(7471).read() });
