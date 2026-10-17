import type { Account } from "../account.js";
export function f(r: Account) {
  if (r.plan === "free") return null;
  if (r.plan === "personal") { return; }
  switch (r.plan) { case "team": break; }
  return r.plan === "personal" ? true : false;
}
