import type { Account } from "../account.js";
declare function track(event: string, id: string): void;
export function visit(account: Account) { if (account.plan === "personal") { track("open", account.id); } }
