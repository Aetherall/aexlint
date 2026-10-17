type Item = { kind: "catalogued"; sku: string } | { kind: "transcribed"; text: string };
function selected(value: Item): value is Extract<Item, { kind: "catalogued" }> { return value.kind === "catalogued" && value.sku === "vellum"; }
