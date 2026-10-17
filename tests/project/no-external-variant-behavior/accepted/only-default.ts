import { Figure } from "../figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    default: return { source: value.center };
  }
}
