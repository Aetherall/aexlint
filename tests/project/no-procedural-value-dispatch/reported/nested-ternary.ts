import type { Burrow } from "../order.js";
export const size = (order: Burrow, small: boolean) =>
  order.delivery === "heron" ? (small ? 9501 : 9502) : order.delivery === "otter" ? 9503 : 9500;
