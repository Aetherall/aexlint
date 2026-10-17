export namespace Tokens {
export type Token = { category: "tram"; text: string } | { category: "ferry"; amount: number };
export type TokenAlias = Token;
export function transform(value: Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
}
export namespace Tokens {
export function second(value: Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
}
