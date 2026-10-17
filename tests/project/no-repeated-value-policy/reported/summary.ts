import type { Account } from "../account.js";
export function summary(accounts: Account[], fallback: string[]) {
  return accounts.map((account) => ({ allowed: account.plan === "free" ? ["basic"] : fallback }));
}
