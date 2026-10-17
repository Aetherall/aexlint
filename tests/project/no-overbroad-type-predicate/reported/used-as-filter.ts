type Item = { kind: "archived" | "lent" };
function selected(value: Item): value is Item { return value.kind === "archived"; }
declare const values: Item[];
const chosen = values.filter(selected);
