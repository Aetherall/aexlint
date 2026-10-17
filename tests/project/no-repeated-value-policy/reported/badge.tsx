import type { Account } from "../account.js";
export const Badge = (props: { r: Account }) => <div>{props.r.plan === "free" && <span>free</span>}</div>;
