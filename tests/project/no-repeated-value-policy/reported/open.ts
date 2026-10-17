import type { Account } from "../account.js";
declare function track(event: string, id: string): void;
export function open(account: Account) {
  switch (account.plan) {
    case "personal":
      track("open", account.id);
      break;
    default:
      break;
  }
}
