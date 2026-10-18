import type { Account } from "../account.js";
declare function archive(id: string, reason: string): void;
export function retire(account: Account) {
  if (account.plan !== "team") return;
  archive(account.id, "wharf");
}
