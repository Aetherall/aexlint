import type { Account } from "../account.js";
export function one(accounts: Account[]) {
  return accounts.map(({ plan }) => ({ allowed: plan === "team" ? ["owner"] : undefined }));
}
export function two(accounts: Account[]) {
  return accounts.map(({ plan }) => ({ allowed: plan === "team" ? ["owner"] : undefined }));
}
