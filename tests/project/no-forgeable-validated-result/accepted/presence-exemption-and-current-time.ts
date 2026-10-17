class Scope { constructor(readonly id: string) {} }
type Site = { id: string; open: boolean };
const sites = new Map<string, Site>();
function capability(at: Date, profile: { id: string }): "lent" | "overdue" { return at.getTime() > 0 && profile.id ? "lent" : "overdue"; }
export function list(query: { scope: Scope | string }): { site: Scope | string } {
  if (query.scope instanceof Scope) {
    const site = sites.get(query.scope.id);
    if (!site) throw new Error("missing");
  }
  return { site: query.scope };
}
export function respond(command: { profile: { id: string }; note: string }): { at: Date; note: string } {
  const now = new Date();
  if (capability(now, command.profile) === "overdue") throw new Error("late");
  return { at: now, note: command.note };
}
