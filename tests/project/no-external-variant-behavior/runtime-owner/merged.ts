import type { TokenAlias } from "./token.js";
export function transform(value: TokenAlias) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
