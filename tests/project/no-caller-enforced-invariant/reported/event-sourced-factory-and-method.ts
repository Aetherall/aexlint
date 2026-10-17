class Aggregate { protected readonly events: unknown[] = []; protected record(event: unknown): void { this.events.push(event); } }
export class Token extends Aggregate {
  static issue(expiresAt: Date): { token: Token; secret: string } { const token = new Token(); token.record({ expiresAt }); return { token, secret: "s" }; }
  extend(until: Date): void { this.record({ until }); }
}
export function issue(input: { expiresAt: Date }, token: Token) {
  if (input.expiresAt <= new Date()) throw new Error("past");
  token.extend(input.expiresAt);
  return Token.issue(input.expiresAt);
}
