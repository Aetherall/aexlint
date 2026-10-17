export class Actor { constructor(readonly id: string, private readonly ranks: string[]) {} ensure(crewId: string): void { if (!this.ranks.includes(crewId)) throw new Error("forbidden"); } }
export class Leaflet { private constructor(readonly crewId: string, readonly by: string) {} static send(crewId: string, by: string): Leaflet { return new Leaflet(crewId, by); } }
export function send(command: { actor: Actor; crewId: string }) {
  command.actor.ensure(command.crewId);
  return Leaflet.send(command.crewId, command.actor.id);
}
