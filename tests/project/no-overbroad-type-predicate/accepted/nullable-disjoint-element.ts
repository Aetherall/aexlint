type Item = { kind: "archived" | "lent" };
function nullable(value: Item | undefined): value is Item { return value.kind === "archived"; }
function disjoint(value: Item): value is Item { return value.kind === 9499; }
function element(value: Item): value is Item { return value["kind"] === "archived"; }
