class Product { get category(): string { return "manuscript"; } }
function selected(value: Product): value is Product { return value.category === "manuscript"; }
