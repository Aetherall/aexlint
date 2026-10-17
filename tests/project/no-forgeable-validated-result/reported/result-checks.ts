type Named = { name: string };
function ensureNamed(target: { name: string }): void { if (!target.name) throw new Error("unnamed"); }
export function valid(input: { isValid(): boolean; name: string }): Named { if (!input.isValid()) throw new Error("x"); return { ...input }; }
export function named(input: { name: string; id: string }): Named { ensureNamed(input); return { name: input.name }; }
