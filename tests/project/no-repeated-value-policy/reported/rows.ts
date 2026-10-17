import type { Account } from "../account.js";
export function rows(accounts: Account[]) {
  return accounts.map(({ id, plan }) => ({ id, allowed: plan === "free" ? ["basic"] : undefined }));
}
