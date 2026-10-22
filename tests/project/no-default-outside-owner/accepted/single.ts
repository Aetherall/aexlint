export interface Lamp {
  readonly glow?: "lamp-dim" | "lamp-bright";
  readonly label: string;
  readonly hue?: string;
}

export function glow(lamp: Lamp): string {
  return lamp.glow ?? "lamp-dim";
}

export function described(lamp: Lamp, other: { hue?: string }): unknown[] {
  return [lamp.label ?? "lamp-none", lamp.hue ?? null, lamp.hue ?? 7, other.hue ?? "lamp-red", (lamp as any).glow ?? "lamp-x"];
}

export function unresolved(first: Missing, second: Missing): unknown[] {
  return [first.glow ?? "lamp-dim", second.glow ?? "lamp-bright"];
}
