export class Inspection { private constructor(readonly farmId: string) {} static old(farmId: string): Inspection { return new Inspection(farmId); } }
export function countersign(command: { actor: { id: string; root: boolean }; farmId: string }) {
  if (!command.actor.root && command.actor.id !== command.farmId) throw new Error("other farm");
  return Inspection.old(command.farmId);
}
