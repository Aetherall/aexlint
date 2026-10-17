export type Token = { category: "tram"; text: string } | { category: "ferry"; amount: number };
export type TokenAlias = Token;
type Local = Token;
export function transform(value: Local) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
