import type { Account } from "../account.js";
export function other(accounts: Account[]) {
  return accounts.map(({ plan }) => ({ allowed: plan === "team" ? ["paid"] : undefined, named: plan === "free" ? ["seats"] : undefined }));
}
