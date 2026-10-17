import type { Account } from "../account.js";
declare function track(event: string, id: string): void;
export function pick(account: Account) {
  if (account.plan !== "free") {
    track("select", account.id);
  }
}
