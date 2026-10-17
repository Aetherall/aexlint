import type { Token } from "../token.js";
export function transform(holder: { value: Token }) {
  switch (holder.value.category) {
    case "tram": return { value: holder.value.text };
    case "ferry": return { value: holder.value.amount };
  }
}
