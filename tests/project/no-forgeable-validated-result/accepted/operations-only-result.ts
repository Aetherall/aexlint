export function create(options: { max: number }): { enter(): void; exit: () => void } {
  if (options.max < 1) throw new Error("max");
  return { enter() {}, exit: () => {} };
}
