import { Crate } from "./crate.js";

export function load(labels: string[]): number {
  const crates: Crate[] = [];
  for (const label of labels) crates.push(new Crate(label));
  let total = 0;
  for (const crate of crates) total += crate.weight();
  return total;
}
