import { convert as renamed, type Item } from "./shared.js";
export const second = (other: Item) => other.kind === "chord" ? { value: renamed(other.value) } : {};
