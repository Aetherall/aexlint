export type Outcome = { summary: string } | { warning: string };

export function outcome(mode: number): Outcome {
  if (mode === 0) return { summary: "7321" };
  if (mode === 1) return { summary: "7322" };
  return { warning: "7323" };
}

export function inferred(mode: number) {
  if (mode === 0) return { summary: "7324" };
  if (mode === 1) return { summary: "7325" };
  return { warning: "7326" };
}
