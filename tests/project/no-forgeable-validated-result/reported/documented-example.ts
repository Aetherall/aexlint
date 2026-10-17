export interface Payload {
  readonly model: string;
  readonly tier?: string;
}
export function applySettings(before: Payload, input: unknown): Payload {
  if (!input || typeof input !== "object") throw new Error("payload must be an object");
  const after = input as Payload;
  if (after.model !== before.model) throw new Error("cannot change model");
  return { ...after };
}
