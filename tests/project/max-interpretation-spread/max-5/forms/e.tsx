import type { Item } from "./kinds.js"; export const J = (props: { item: Item }) => <div>{props.item.kind === "brume" ? "brume" : ""}</div>;
