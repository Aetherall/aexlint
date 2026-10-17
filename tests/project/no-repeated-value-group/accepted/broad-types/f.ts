export function f(s: string, u: unknown, x: any, m: Missing) {
  if (s === "jackal" || s === "hyena") return 9501;
  if (u === "jackal" || u === "hyena") return 9502;
  if (x === "jackal" || x === "hyena") return 9503;
  if (m === "jackal" || m === "hyena") return 9504;
  return s === "jackal" || s === "hyena" ? 9505 : 9506;
}
