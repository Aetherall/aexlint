interface Token { category: string; text: string; amount: number }
export function transform(value: Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
