import { create } from "./token.js";
export function transform() {
  const value = create();
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
