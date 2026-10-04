import { ruleTester } from "../../../tests/rule-tester.ts";
import rule from "./index.ts";

ruleTester.run("no-else-if", rule, {
  valid: [
    "if (a) { run(); }",
    "if (a) { run(); } else { stop(); }",
    "if (a) run(); else stop();",
    "if (a) return; if (b) return;",
    "if (a) { run(); } else { if (b) log(); run(); }",
    "if (a) { run(); } else {\n  if (b) { log(); }\n}",
    "const label = a ? 'a' : b ? 'b' : 'c';",
    "if (a) { if (b) { run(); } else { stop(); } }",
    { code: "if (props.open) { render(<div />); } else { close(); }", filename: "view.tsx" },
    { code: "if (a) {} else {}", filename: "input.js" },
  ],
  invalid: [
    {
      code: "if (a) { run(); } else if (b) { stop(); }",
      errors: [{ messageId: "elseIf", line: 1, column: 18, endLine: 1, endColumn: 25 }],
      output: null,
    },
    {
      code: "if (a) {\n  one();\n} else if (b) {\n  two();\n} else if (c) {\n  three();\n} else {\n  four();\n}",
      errors: [
        { messageId: "elseIf", line: 3, column: 2, endLine: 3, endColumn: 9 },
        { messageId: "elseIf", line: 5, column: 2, endLine: 5, endColumn: 9 },
      ],
      output: null,
    },
    {
      code: "function f() {\n  if (a) return 1;\n  else if (b) return 2;\n  return 3;\n}",
      errors: [{ messageId: "elseIf", line: 3, column: 2, endLine: 3, endColumn: 9 }],
      output: null,
    },
    {
      code: "if (a) {} else /* fallback */ if (b) {}",
      errors: [{ messageId: "elseIf", line: 1, column: 10, endLine: 1, endColumn: 32 }],
      output: null,
    },
    {
      code: "if (a) {} else\nif (b) {}",
      errors: [{ messageId: "elseIf", line: 1, column: 10, endLine: 2, endColumn: 2 }],
      output: null,
    },
    {
      code: "function f() { return () => { if (a) { return 1; } else if (b) { return 2; } return 3; }; }",
      errors: [{ messageId: "elseIf" }],
      output: null,
    },
    {
      code: "if (props.open) { render(<div />); } else if (props.closing) { fade(); }",
      filename: "view.tsx",
      errors: [{ messageId: "elseIf" }],
      output: null,
    },
    {
      code: "if (a) {} else if (b) {}",
      filename: "input.js",
      errors: [{ messageId: "elseIf" }],
      output: null,
    },
  ],
});
