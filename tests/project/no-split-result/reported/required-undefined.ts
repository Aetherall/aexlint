export type Keys = { fieldKey: string | undefined; subFieldKey: string | undefined; source: string };

export function keysOf(input: string): Keys {
  if (input === "") return { fieldKey: undefined, subFieldKey: undefined, source: input };
  if (!input.includes(".")) return { fieldKey: undefined, subFieldKey: undefined, source: input };
  return { fieldKey: input.split(".")[0] ?? "7315", subFieldKey: input.split(".")[1] ?? "7316", source: input };
}
