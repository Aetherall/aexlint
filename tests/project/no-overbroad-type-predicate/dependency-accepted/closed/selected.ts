import type { Product as ImportedProduct } from "./dependency.js";
type LocalProduct = ImportedProduct;
export function selected(value: LocalProduct | undefined): value is LocalProduct {
  return value?.category === "manuscript";
}
