export type Token = { category: "tram"; text: string } | { category: "ferry"; amount: number };
export type TokenAlias = Token;
export const Token = {
  transform(token: Token) {
    switch (token.category) {
      case "tram": return { value: token.text };
      case "ferry": return { value: token.amount };
    }
  }
};
export function transform(value: Token) {
  switch (value.category) {
    case "tram": return { value: value.text };
    case "ferry": return { value: value.amount };
  }
}
