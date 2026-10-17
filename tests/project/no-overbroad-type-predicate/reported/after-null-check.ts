type Product = { category: "manuscript" | "folio" };
function selected(value: Product | null | undefined): value is Product { return value != null && value.category === "manuscript"; }
