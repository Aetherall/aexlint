type Item = { kind: "archived" | "lent" };
function first(value: Item): value is Item { return value.kind === "archived"; }
const second = (other: Item): other is Item => other.kind !== "lent";
