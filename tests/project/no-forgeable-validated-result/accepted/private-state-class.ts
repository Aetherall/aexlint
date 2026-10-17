export class Payload {
  readonly #model: string;
  constructor(input: unknown) {
    if (typeof input !== "string") throw new Error("model");
    this.#model = input;
  }
}
export function parsePayload(input: unknown): Payload { return new Payload(input); }
export class Guarded {
  private readonly model: string;
  constructor(input: string) { if (!input) throw new Error("model"); this.model = input; }
}
