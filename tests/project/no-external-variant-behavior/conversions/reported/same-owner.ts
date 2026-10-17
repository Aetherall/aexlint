import type { Token } from "../token.js";
export function swap(value: Token): Token {
  return value.category === "tram" ? { category: "ferry", amount: value.text.length } : { category: "tram", text: String(value.amount) };
}
