import { convert, type Item } from "./shared.js";
export const first = (item: Item) => item.kind === "chord" ? { value: convert(item.value) } : {};
