import { Credential } from "./credential.js";
type Actor = { id: string; ranks: string[] };
function can(actor: Actor, action: string): boolean { return actor.ranks.includes(action); }
export class Grant { private constructor(readonly actorId: string) {} static of(actorId: string): Grant { return new Grant(actorId); } }
export function grant(command: { actor: Actor }) {
  if (!can(command.actor, "grant")) throw new Error("forbidden");
  return Grant.of(command.actor.id);
}
