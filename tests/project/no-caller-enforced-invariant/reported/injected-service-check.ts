export class Ranger { private constructor(readonly crewId: string) {} static enlist(crewId: string): Ranger { return new Ranger(crewId); } }
export class Handler {
  constructor(private readonly registrations: { enabled(crewId: string): boolean }) {}
  run(command: { crewId: string }) {
    if (!this.registrations.enabled(command.crewId)) throw new Error("not available");
    return Ranger.enlist(command.crewId);
  }
}
