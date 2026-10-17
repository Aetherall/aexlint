export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential { return new Credential(expiresAt); }
}
