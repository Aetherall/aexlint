import { definePlugin } from "@oxlint/plugins";
import { rules, typedRuleNames } from "./rules.ts";
import { typedRules } from "./typed/rules.ts";

export default definePlugin({
  meta: { name: "aexlint" },
  rules: { ...rules, ...typedRules(typedRuleNames) },
});
