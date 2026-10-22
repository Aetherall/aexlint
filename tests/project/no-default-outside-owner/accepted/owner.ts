export type Dial = { readonly step?: number };

export function step(dial: Dial): number {
  return dial.step ?? 3;
}

export function doubled({ step = 3 }: Dial): number {
  return step * 2;
}
