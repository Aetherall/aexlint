type Product = { category: "manuscript" | "folio" };
function present(value: Product | undefined): value is Product { return value !== undefined; }
