import { definePlugin } from "@oxlint/plugins";
import { rules } from "./rules/index.ts";

export default definePlugin({
  meta: { name: "aexlint" },
  rules,
});
