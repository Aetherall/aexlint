export class Actor { constructor(readonly userId: string, private readonly crews: string[]) {} canAccessCrew(crewId: string): boolean { return this.crews.includes(crewId); } }
export class Site { constructor(readonly id: string, readonly crewId: string) {} ensureMayAccess(actor: Actor): void { if (!actor.canAccessCrew(this.crewId)) throw new Error("forbidden"); } }
export class Leaflet { private constructor(readonly site: Site, readonly by: string, readonly note: string) {} static send(input: { site: Site; by: string; note: string }): Leaflet { return new Leaflet(input.site, input.by, input.note); } }
const sites = new Map<string, Site>();
export function send(command: { actor: Actor; siteId: string }) {
  const { actor, siteId } = command;
  const site = sites.get(siteId)!;
  site.ensureMayAccess(actor);
  return Leaflet.send({ site, by: actor.userId, note: "été" });
}
