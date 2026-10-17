type Product<T> = { category: "manuscript" | "folio"; data: T };
function selected(value: Product<any> | undefined): value is Product<any> { return value?.category === "manuscript"; }
