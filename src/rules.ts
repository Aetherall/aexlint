import type { Rule } from "@oxlint/plugins";
import rule0 from "../rules/max-call-assembly/rule.ts";
import rule1 from "../rules/max-decision-depth/rule.ts";
import rule2 from "../rules/max-expression-complexity/rule.ts";
import rule3 from "../rules/max-expression-depth/rule.ts";
import rule4 from "../rules/max-held-context/rule.ts";
import rule5 from "../rules/no-else-if/rule.ts";
import rule6 from "../rules/no-multiline-condition/rule.ts";
import rule7 from "../rules/prefer-domain-predicate/rule.ts";

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

export const typedRuleNames: string[] = [
  "max-interpretation-spread",
  "max-projection-spread",
  "no-caller-enforced-invariant",
  "no-external-variant-behavior",
  "no-fact-outside-owner",
  "no-forgeable-validated-result",
  "no-overbroad-type-predicate",
  "no-procedural-value-dispatch",
  "no-repeated-conversion",
  "no-repeated-presence-check",
  "no-repeated-selection",
  "no-repeated-value-group",
  "no-repeated-value-policy",
  "prefer-truthy-presence-check",
];
