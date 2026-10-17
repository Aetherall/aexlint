type Product = { category: string };
const helpers = { selected(value: Product): value is Product { return value.category === ""; } };
