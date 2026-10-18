import type { Account } from "../account.js";
declare function archive(id: string, reason: string): void;
export function keepTeam(account: Account) {
  if (account.plan === "team") {
    archive(account.id, "jetty");
  }
}
