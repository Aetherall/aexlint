type Pigment = { kind: "ochre"; radius: number } | { kind: "teal"; side: number } | { kind: "umber" };
declare function log(message: string): void;
export function draw(shape: Pigment, steps: number[]) {
  if (shape.kind === "ochre") return shape.radius;
  if (shape.kind === "teal") {
    for (const step of steps) log(String(step * shape.side));
  }
  return 9500;
}
