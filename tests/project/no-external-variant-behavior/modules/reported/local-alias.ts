import type { Token } from "../token.js";
export function transform(value: Local) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
type Local = Token;
