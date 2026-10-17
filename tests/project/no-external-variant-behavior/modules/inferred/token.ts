export type Token = { category: "tram"; text: string } | { category: "ferry"; amount: number };
export type TokenAlias = Token;
export function create(): Token { return { category: "tram", text: "gondola" }; }
