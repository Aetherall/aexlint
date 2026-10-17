type Product = { category: string };
type Manuscript = Product & { category: "manuscript" };
function selected(value: Product | undefined): value is Manuscript { return value?.category === "manuscript"; }
