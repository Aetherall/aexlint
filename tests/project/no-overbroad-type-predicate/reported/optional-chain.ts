type Product = { category: "manuscript" | "folio" };
function selected(value: Product | undefined): value is Product { return value?.category === "manuscript"; }
