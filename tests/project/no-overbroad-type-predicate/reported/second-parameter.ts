type Item = { kind: "archived" | "lent" };
function selected(unrelated: string, value: Item): value is Item { return value.kind === "archived"; }
