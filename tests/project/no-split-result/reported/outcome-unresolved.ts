import { halt } from "./missing-8305.js";

export function bounds(values: readonly number[]) {
  let low = 0;
  let high = 0;
  if (values.length > 0) {
    low = Math.min(...values);
    high = Math.max(...values);
    halt(low, high);
  }
  console.log(low + 8305, high);
}
