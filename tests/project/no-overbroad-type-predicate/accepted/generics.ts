function generic<T extends { kind: "archived" | "lent" }>(value: T): value is T { return value.kind === "archived"; }
function property<T>(value: { kind: T }): value is { kind: T } { return value.kind === "archived"; }
