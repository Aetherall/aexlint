import { ruleTester } from "../../../tests/rule-tester.ts";
import rule from "./index.ts";

const options = [{ max: 3 }];
const tooDeep = (depth: number, max = 3) => ({
  messageId: "tooDeep",
  data: { depth, open: depth - 1, max },
});

ruleTester.run("max-held-context", rule, {
  valid: [
    ...[
      'function validate(a, b, c) { if (!a) throw new Error("a"); if (!b) throw new Error("b"); if (!c) throw new Error("c"); return build(a, b, c); }',
      'function label(kind) { if (kind === "a") return "A"; else if (kind === "b") return "B"; else if (kind === "c") return "C"; else if (kind === "d") return "D"; return "other"; }',
      "function save(items) { return perform(async () => { await Promise.all(items.map(async (item) => Promise.all([item].map(async (inner) => store.save(inner))))); }); }",
      "function factory(a, b, c) { return define({ update() { if (a) { if (b) { if (c) work(); } } } }); }",
      "function factory(a, b, c, d) { const handler = () => { if (a) { if (b) { if (c) work(); } } }; if (d) handler(); }",
      "class A { m(a, b, c) { if (a) { if (b) { if (c) work(); } } } }",
      "for (const x of xs) { for (const y of x) { if (!y) continue; use(y); } }",
      "for (const x of xs) { if (x) { for (const y of x) use(y); } }",
      "if (a && (b || (c && d))) work();",
      "try { if (a) b(); } catch (error) { if (c) d(); }",
      "const value = items.map((item) => (item ? 1 : 2));",
      "switch (kind) { case 1: if (a) { if (b) work(); } break; default: other(); }",
      "if (a) { work(); } if (b) { if (c) { if (d) {} } }",
      "function f(a) { if (!a) return items.map((x) => (x ? 1 : 2)); return 0; }",
    ].map((code) => ({ code, options })),
    {
      code: "const View = () => <div>{items.map((item) => (item.visible ? <a /> : <b />))}</div>;",
      filename: "view.tsx",
      options,
    },
    { code: "if (a) { if (b) {} }", options: [{ max: 2 }] },
    { code: "if (a) {}", options: [{ max: 1 }] },
  ],
  invalid: [
    {
      code: "if (a) { if (b) { if (c) { if (d) work(); } } }",
      options,
      errors: [{ ...tooDeep(4), line: 1, column: 27, endLine: 1, endColumn: 33 }],
      output: null,
    },
    {
      code: 'function carried(items) { for (const item of items) { if (item.length > 0) { item.map((value) => { if (value < 0) throw new Error("negative"); return value; }); } } }',
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "if (a) { for (const x of xs) { f(() => (x ? 1 : 2)); } }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "switch (kind) { case 1: for (const x of xs) { if (x) { if (y) {} } } }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "for (const x of xs) { try { work(x); } catch (error) { if (a) { if (b) {} } } }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "if (a) { if (b) { if (c) {} else if (d) { if (e) {} } } }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "if (a) { if (b) { if (c) { if (d) { if (e) work(); } } } }",
      options,
      errors: [{ ...tooDeep(4), line: 1, column: 27, endLine: 1, endColumn: 33 }],
      output: null,
    },
    {
      code: "if (a) { if (b) { if (c) { if (d) {} } } } if (e) { if (f) { if (g) { if (h) {} } } }",
      options,
      errors: [tooDeep(4), tooDeep(4)],
      output: null,
    },
    {
      code: "for (const x of xs) {\n  for (const y of x) {\n    for (const z of y) {\n      for (const w of z) use(w);\n    }\n  }\n}",
      options,
      errors: [{ ...tooDeep(4), line: 4, column: 6, endLine: 4, endColumn: 24 }],
      output: null,
    },
    {
      code: "function m(a, b, c, d) { const inner = { run() { if (a) { if (b) { if (c) { if (d) {} } } } } }; return inner; }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "for (const a of as) { for (const b of a) { for (const c of b) { if (!c.x) continue; if (!c.y) continue; if (!c.z) continue; use(c); } } }",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
    {
      code: "if (a) { if (b) { if (c) { if (d) {} } else { if (e) {} } } }",
      options,
      errors: [tooDeep(4), tooDeep(4)],
      output: null,
    },
    {
      code: "if (a) { if (b) {} }",
      options: [{ max: 1 }],
      errors: [tooDeep(2, 1)],
      output: null,
    },
    {
      code: "const View = () => <div>{items.map((item) => (item.visible ? (item.open ? (item.big ? <a /> : <b />) : <c />) : <d />))}</div>;",
      filename: "view.tsx",
      options,
      errors: [tooDeep(4)],
      output: null,
    },
  ],
});
