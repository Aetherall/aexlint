type Swatch = { kind: "mint" } | { kind: "peach" } | { kind: "pearl" } | { kind: "ruby" };
export function f(shape: Swatch) {
  if (shape.kind === "mint") return 9500;
  if (shape.kind === "peach" || shape.kind === "pearl") return 9501;
  return [shape.kind === "pearl" || shape.kind === "peach" ? 9502 : 9503];
}
