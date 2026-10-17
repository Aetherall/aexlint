type Item = { kind: "archived" | "lent" };
function selected(value: Item): value is Item { return value.kind === "archived"; }
