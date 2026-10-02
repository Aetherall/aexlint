import { defineRule } from "@oxlint/plugins";
import { ruleTester } from "./rule-tester.ts";

const rule = defineRule({
  meta: {
    type: "problem",
    schema: [],
    messages: { unexpected: "Remove the debugger statement." },
    fixable: "code",
  },
  create(context) {
    return {
      DebuggerStatement(node) {
        context.report({
          node,
          messageId: "unexpected",
          fix: (fixer) => fixer.remove(node),
        });
      },
    };
  },
});

ruleTester.run("harness/no-debugger", rule, {
  valid: [
    "const value: number = 1;",
    { filename: "example.js", code: 'const value = "debugger;";' },
    { filename: "example.tsx", code: "const element = <div />;" },
  ],
  invalid: [
    {
      code: "debugger;",
      errors: [{ messageId: "unexpected", line: 1, column: 0, endColumn: 9 }],
      output: "",
    },
    {
      code: "debugger; debugger;",
      errors: [{ messageId: "unexpected" }, { messageId: "unexpected" }],
      output: " ",
    },
  ],
});
