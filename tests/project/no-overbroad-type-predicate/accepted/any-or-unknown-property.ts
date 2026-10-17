type Item = { kind: any } | { kind: unknown };
function selected(value: Item): value is Item { return value.kind === "archived"; }
