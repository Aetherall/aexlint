type Item = { kind: "archived" | "lent" };
function defaulted(value: Item, ignored = (value.kind = "archived")): value is Item { return value.kind === "archived"; }
function destructured(value: Item, { [value.kind = "archived"]: ignored }: { archived?: unknown }): value is Item { return value.kind === "archived"; }
function harmlessDefault(value: Item, ignored = 9403): value is Item { return value.kind === "archived"; }
function harmlessBinding(value: Item, { ignored }: { ignored: number }): value is Item { return value.kind === "archived"; }
