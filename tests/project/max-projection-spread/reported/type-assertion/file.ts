export interface Gauge { at(slot: number): { read(): number } }
export const a = (gauge: Gauge) => ({ low: gauge.at(7470).read(), high: gauge.at(7471).read() });
