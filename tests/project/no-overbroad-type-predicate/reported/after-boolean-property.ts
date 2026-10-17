type Item = { kind: "archived" | "lent"; ready: boolean };
function selected(value: Item): value is Item { return value.ready && value.kind === "archived"; }
