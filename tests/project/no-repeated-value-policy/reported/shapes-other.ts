import type { Account, Plan } from "../account.js";
type Tier = Plan;
declare function grant(scope: string): void;
export function aliased(tier: Tier) { if (tier === "team") grant("vault"); }
export function generic<T extends Account>(account: T) { if (account.plan === "personal") grant("cellar"); }
export function optional(plan: Plan | undefined) { if (plan === "team") grant("attic"); }
