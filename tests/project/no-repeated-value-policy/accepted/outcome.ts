import type { Account } from "../account.js";
export function trial(accounts: Account[]) {
  return accounts.map(({ plan }) => ({ allowed: plan === "team" ? ["trial"] : undefined, named: plan === "personal" ? ["seats"] : undefined }));
}
