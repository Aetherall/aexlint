export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential {
    if (expiresAt <= new Date()) throw new Error("past");
    return new Credential(expiresAt);
  }
}
