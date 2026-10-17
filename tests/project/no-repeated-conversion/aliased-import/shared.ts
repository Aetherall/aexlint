export type Item = { kind: "chord" | "tangent"; value: string };
export const convert = (value: string) => value.toUpperCase();
