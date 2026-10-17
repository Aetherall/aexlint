type Item = { kind: "archived" | "lent" };
function union(value: Item): value is Item { return value.kind === "archived" || value.kind === "lent"; }
declare function normalize(value: Item): boolean;
function effect(value: Item): value is Item { return normalize(value) && value.kind === "archived"; }
function assign(value: Item): value is Item { return (value.kind = "archived") === "archived" && value.kind === "archived"; }
