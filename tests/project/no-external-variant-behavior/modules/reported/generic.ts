import type { Token } from "../token.js";
export function transform<T extends Token>(value: T) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
