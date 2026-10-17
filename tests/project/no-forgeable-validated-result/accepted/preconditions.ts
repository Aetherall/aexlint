type Template = { id: string; isDefault: boolean };
type Actor = { ranks: string[] };
const templates = new Map<string, Template>();
const existing: Template[] = [];
function ensureChief(actor: Actor): void { if (!actor.ranks.includes("chief")) throw new Error("forbidden"); }
export function rename(command: { id: string; actor: Actor }): { id: string } {
  ensureChief(command.actor);
  const template = templates.get(command.id)!;
  if (template.isDefault) throw new Error("default");
  return { id: template.id };
}
export function create(command: { id: string; name: string }): { id: string; name: string } {
  const duplicates = existing.filter((template) => template.id === command.id);
  if (duplicates.length > 0) throw new Error("duplicate");
  return { id: command.id, name: command.name };
}
