type Item = { kind: "archived" | "lent" };
declare function external(value: Item): boolean;
function selected(value: Item): value is Item { return value.kind === "archived" && external(value); }
