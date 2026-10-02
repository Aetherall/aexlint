import { describe, it } from "node:test";
import { RuleTester } from "oxlint/plugins-dev";

RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;

export const ruleTester = new RuleTester({
  languageOptions: { parserOptions: { lang: "ts" } },
});
