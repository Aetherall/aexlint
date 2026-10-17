import { Tokens } from "./token.js";
export function transform(value: Tokens.Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
