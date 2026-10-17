type Stand = { height: number; tilt: number };
type Adjustment = { change: "raise" | "drop"; stand: Stand; step: number };

export function adjust({ change, stand, step }: Adjustment): Stand {
  let height = stand.height;
  if (change === "raise") height = stand.height + step;
  return { ...stand, height };
}
