export interface Token { category: "tram" | "ferry"; text: string; amount: number }
export function transform(value: Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
