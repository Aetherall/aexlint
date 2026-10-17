import { Figure } from "../figure.js";
import type { Label } from "../label.js";
import type { Token } from "../token.js";
import { Shape } from "../shape.js";
export function many(value: Figure, ready: boolean) {
  switch (value.kind) {
    case "polygon": {
      const label: Label = { text: value.start };
      return label;
    }
    case "ellipse": {
      if (ready) {
        const token: Token = { category: "ferry", amount: value.center };
        return token;
      }
      return new Shape({ size: value.center });
    }
  }
}
