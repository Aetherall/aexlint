type Payload = { model: string };
export function d(input: string): Payload { if (input) return { model: input }; throw new Error("empty"); }
