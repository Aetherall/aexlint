export class Token { static generate() { return new Token(); } value() { return 1; } }
export const a = () => ({ id: Token.generate().value(), from: Token.generate().value() });
