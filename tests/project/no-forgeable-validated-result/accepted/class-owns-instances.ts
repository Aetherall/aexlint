export class Settings {
  model: string;
  constructor(input: unknown) {
    if (typeof input !== "string") throw new Error("model");
    this.model = input;
  }
}
export class Name {
  constructor(readonly value: string) {}
  static parse(input: string): Name { if (!input) throw new Error("empty"); return new Name(input); }
}
class Check { constructor(input: string) { if (!input) throw new Error("x"); } }
type Payload = { model: string };
export function make(input: string): Payload { new Check(input); return { model: input }; }
