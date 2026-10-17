class Actor { constructor(readonly id: string, readonly ranks: string[]) {} }
type Entity = { id: string; crewId: string; name: string; updatedAt: string };
function ensureRanger(actor: Actor): void { if (!actor.ranks.includes("ranger")) throw new Error("forbidden"); }
function load(id: string): Entity { return { id, crewId: "", name: "", updatedAt: "" }; }
export function folder(command: { folder: { ids(): string[]; siteId: string } }): { siteId: string } {
  if (!command.folder.ids().includes("shelfmark")) throw new Error("missing");
  return { siteId: command.folder.siteId };
}
export function ranger(actor: Actor): { actorId: string } { ensureRanger(actor); return { actorId: actor.id }; }
export function edit(command: { id: string; name: string; crewId: string }): { updatedAt: string } {
  const entity = load(command.id);
  entity.name = command.name;
  if (entity.crewId !== command.crewId) throw new Error("other crew");
  return { updatedAt: entity.updatedAt };
}
