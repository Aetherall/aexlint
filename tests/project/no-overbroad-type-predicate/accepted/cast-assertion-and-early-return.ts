type Item = { kind: "archived" | "lent" };
function cast(value: Item): value is Item { return (value.kind as "archived") === "archived"; }
function asserted(value: Item): asserts value is Item { if (value.kind !== "archived") throw new Error(); }
function ladder(value: Item | undefined): value is Item { if (!value) return false; return value.kind === "archived"; }
