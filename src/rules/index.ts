import type { Rule } from "@oxlint/plugins";
import rule0 from "./max-call-assembly/index.ts";
import rule1 from "./max-decision-depth/index.ts";
import rule2 from "./max-expression-complexity/index.ts";
import rule3 from "./max-expression-depth/index.ts";
import rule4 from "./max-held-context/index.ts";
import rule5 from "./no-else-if/index.ts";
import rule6 from "./no-multiline-condition/index.ts";
import rule7 from "./prefer-domain-predicate/index.ts";

export const rules: Record<string, Rule> = {
  "max-call-assembly": rule0,
  "max-decision-depth": rule1,
  "max-expression-complexity": rule2,
  "max-expression-depth": rule3,
  "max-held-context": rule4,
  "no-else-if": rule5,
  "no-multiline-condition": rule6,
  "prefer-domain-predicate": rule7,
};
