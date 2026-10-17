import type { Account } from "../account.js";
export const Row = ({ r }: { r: Account }) => <p>{r.plan === "free" && <span>free</span>}</p>;
