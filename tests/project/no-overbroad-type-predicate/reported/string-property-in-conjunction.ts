type Product = { category: string; format: string };
function selected(value: Product | undefined): value is Product { return value?.category === "manuscript" && value.format === "bound"; }
