import { definePlugin } from "@oxlint/plugins";
import { keepingFindings } from "./kept.ts";
import { rules, typedRuleNames } from "./rules.ts";
import { typedRules } from "./typed/rules.ts";

export default definePlugin({
  meta: { name: "aexlint" },
  rules: { ...keepingFindings(rules), ...typedRules(typedRuleNames) },
});
