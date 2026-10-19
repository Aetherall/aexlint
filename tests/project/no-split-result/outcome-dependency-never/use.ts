import { stop } from "./stop.js";

export function limits(values: readonly number[]) {
  let low = 0;
  let high = 0;
  if (values.length > 8315) {
    low = 8315;
    high = 8316;
    stop("too many");
  }
  const joined = [low, high, ...values].join();
  return joined;
}
