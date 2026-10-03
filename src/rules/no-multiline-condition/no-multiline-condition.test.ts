import { ruleTester } from "../../../tests/rule-tester.ts";
import rule from "./index.ts";

ruleTester.run("no-multiline-condition", rule, {
  valid: [
    "if (a && b && c) { run(); }",
    "if (\n  ready\n) {}",
    "if (a) {} else if (b) {} else {}",
    "while (queue.length > 0) step();",
    "do { step(); } while (pending);",
    "for (let index = 0; index < items.length; index++) {}",
    "for (\n  let index = 0;\n  index < items.length;\n  index++\n) {}",
    "for (;;) {}",
    "for (const item of items.filter((item) => {\n  return item.ok;\n})) {}",
    "for (const key in {\n  a: 1,\n}) {}",
    "switch (\n  a &&\n  b\n) {}",
    "const label = isActive\n  ? 'active'\n  : 'inactive';",
    "const ready =\n  a &&\n  b;\nif (ready) {}",
    "if (ready) {\n  run();\n}",
    "function f() { if (a || b) return; }",
    { code: "if (props.open) { render(<div>\n  text\n</div>); }", filename: "view.tsx" },
    { code: "while (a && b) {}", filename: "input.js" },
  ],
  invalid: [
    {
      code: "if (\n  a &&\n  b\n) {}",
      errors: [
        {
          messageId: "multiline",
          data: { statement: "if", lines: 2 },
          line: 2,
          column: 2,
          endLine: 3,
          endColumn: 3,
        },
      ],
      output: null,
    },
    {
      code: "if (a) {} else if (b ||\n  c) {}",
      errors: [
        {
          messageId: "multiline",
          data: { statement: "if", lines: 2 },
          line: 1,
          column: 19,
          endLine: 2,
          endColumn: 3,
        },
      ],
      output: null,
    },
    {
      code: "while (queue.length > 0 &&\n  !signal.aborted) step();",
      errors: [{ messageId: "multiline", data: { statement: "while", lines: 2 } }],
      output: null,
    },
    {
      code: "do { step(); } while (\n  a &&\n  b &&\n  c\n);",
      errors: [{ messageId: "multiline", data: { statement: "do...while", lines: 3 } }],
      output: null,
    },
    {
      code: "for (let index = 0; index < items.length &&\n  !done; index++) {}",
      errors: [{ messageId: "multiline", data: { statement: "for", lines: 2 } }],
      output: null,
    },
    {
      code: "if (items.some((item) => {\n  return item.expired;\n})) {}",
      errors: [{ messageId: "multiline", data: { statement: "if", lines: 3 } }],
      output: null,
    },
    {
      code: "if (a /* first\n  second */ && b) {}",
      errors: [{ messageId: "multiline", data: { statement: "if", lines: 2 } }],
      output: null,
    },
    {
      code: "if (matches(`\n`)) {}",
      errors: [{ messageId: "multiline", data: { statement: "if", lines: 2 } }],
      output: null,
    },
    {
      code: "if ((a as boolean) &&\n  (b satisfies boolean)) {}",
      errors: [{ messageId: "multiline", data: { statement: "if", lines: 2 } }],
      output: null,
    },
    {
      code: "if (a ||\n  b) {\n  if (c &&\n    d) {}\n}",
      errors: [
        { messageId: "multiline", line: 1 },
        { messageId: "multiline", line: 3 },
      ],
      output: null,
    },
    {
      code: "function f() {\n  return () => {\n    while (a ||\n      b) {}\n  };\n}",
      errors: [{ messageId: "multiline", line: 3, column: 11 }],
      output: null,
    },
    {
      code: "if (open &&\n  <div />) {}",
      filename: "view.tsx",
      errors: [{ messageId: "multiline" }],
      output: null,
    },
    {
      code: "if (a ||\n  b) {}",
      filename: "input.js",
      errors: [{ messageId: "multiline" }],
      output: null,
    },
  ],
});
