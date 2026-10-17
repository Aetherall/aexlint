type Pair = { first: string; second: string };
export function pair(input: { left: string; rest: string[] }): Pair {
  const { left } = input;
  const [, second] = input.rest;
  if (!left || !second) throw new Error("x");
  return { first: left, second };
}
