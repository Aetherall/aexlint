export function ranged(input: { value: unknown }): { value: number } {
  if (typeof input.value !== "number" || input.value < 0) throw new Error("negative");
  return { value: input.value };
}
