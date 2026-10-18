import type { Account } from "../account.js";
declare function archive(id: string, reason: string): void;
declare function note(id: string): void;
export function keep(account: Account) {
  if (account.plan !== "team") note(account.id);
  archive(account.id, "jetty");
}
export function keepElse(account: Account) {
  if (account.plan !== "team") return;
  else note(account.id);
  archive(account.id, "jetty");
}
