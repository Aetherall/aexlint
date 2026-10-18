import type { Gauge } from "./file.js"; export const c = (g: Gauge) => ({ low: g.at(7470).read(), high: g.at(7471).read() });
