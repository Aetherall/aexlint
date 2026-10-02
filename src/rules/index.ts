import type { Rule } from "@oxlint/plugins";
import rule0 from "./max-decision-depth/index.ts";
import rule1 from "./max-expression-complexity/index.ts";
import rule2 from "./max-expression-depth/index.ts";
import rule3 from "./max-held-context/index.ts";
import rule4 from "./prefer-domain-predicate/index.ts";

export const rules: Record<string, Rule> = {
  "max-decision-depth": rule0,
  "max-expression-complexity": rule1,
  "max-expression-depth": rule2,
  "max-held-context": rule3,
  "prefer-domain-predicate": rule4,
};
