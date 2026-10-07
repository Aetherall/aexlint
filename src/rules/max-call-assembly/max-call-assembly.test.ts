import { ruleTester } from "../../../tests/rule-tester.ts";
import rule from "./index.ts";

const limits = { maxWidth: 20, maxReferences: 2 };
const options = [limits];

const report = (width: number, names: string[], configured = limits) => ({
  messageId: "tooAssembled",
  data: { width, count: names.length, names: names.join(", "), ...configured },
});
const four = ["alpha", "beta", "gamma", "delta"];
const inFunction = (code: string) => `function scope() {\n${code}\n}`;

ruleTester.run("max-call-assembly", rule, {
  valid: [
    ...[
      "f();",
      "new Thing;",
      "f(alpha, beta);",
      "f(alpha, beta, gamma);",
      "f(config.port, config.host, env.name);",
      "f(alpha, alpha, beta, beta, alpha, beta);",
      'f("a very long string literal that goes on and on", alpha);',
      "f(`a template ${alpha} that is long ${beta} enough`);",
      'run("name", rule, {\n  valid: ["first long code sample", "second long code sample"],\n  invalid: [{ code: "third", options }],\n});',
      "items.map((item) => {\n  return combine(item, alpha, beta);\n});",
      "items.forEach(function (item) {\n  record(item, alpha, beta);\n});",
      "f((item, index, collection) => {\n  use(item);\n});",
      "items.reduce((total, item, index) => total + item.amount * index);",
      'on("click", { once: true, handler: () => {\n  close(alpha, beta, gamma);\n} });',
      "f({ run() { use(alpha, beta, gamma); } });",
      "f(class { method() { return [alpha, beta, gamma]; } });",
      "averyveryverylongreceiver.withAVeryLongMethodName(alpha, beta);",
      "source\n  .first(alpha)\n  .second(beta)\n  .third(gamma);",
      "f<VeryLongTypeArgumentName>(alpha, beta);",
      "tag`${alpha} ${beta} ${gamma} ${delta}`;",
      "import(`./${alpha}/${beta}/${gamma}.js`);",
      "f(alpha as VeryLongTypeName, beta satisfies AnotherLongTypeName, (item: Item) => item);",
    ].map((code) => ({ code: inFunction(code), options })),
    ...[
      "items.map((item) => {\n  const normalized = normalize(item);\n  return encode(normalized, options);\n});",
      'on("click", { once: true, handler: () => close(dialog) });',
      "export const styles = StyleSheet.create({\n  box: { color: Color.primary, margin: Spacing.small, radius: Radius.medium },\n});",
      "const visible = useMemo(() => filter(items), [items, query, sortOrder, filters, locale]);",
    ].map((code) => ({ code: inFunction(code), options: [{ maxWidth: 60, maxReferences: 3 }] })),
    {
      code: inFunction('f("a string long enough to be wide on its own", alpha, beta, gamma);'),
      options: [{ maxWidth: 24, maxReferences: 2 }],
    },
    {
      code: inFunction("f(`a long template ${alpha} with text ${beta} around ${gamma}`);"),
      options: [{ maxWidth: 27, maxReferences: 2 }],
    },
    {
      code: inFunction("f(alpha, beta, gamma, delta);"),
      options: [{ maxWidth: 27, maxReferences: 2 }],
    },
    {
      code: inFunction("f(alpha, beta, gamma, delta);"),
      options: [{ maxWidth: 20, maxReferences: 4 }],
    },
    {
      code: inFunction("render(<Panel title={heading} open>\n  long text content here\n</Panel>);"),
      filename: "view.tsx",
      options,
    },
    { code: inFunction("f(alpha, beta, gamma);"), filename: "input.js", options },
  ],
  invalid: [
    {
      code: inFunction("f(alpha, beta, gamma, delta);"),
      options,
      errors: [
        {
          ...report(27, four),
          line: 2,
          column: 1,
          endLine: 2,
          endColumn: 28,
        },
      ],
      output: null,
    },
    {
      code: inFunction("f(alpha, beta, gamma, delta);"),
      options: [{ maxWidth: 26, maxReferences: 3 }],
      errors: [report(27, four, { maxWidth: 26, maxReferences: 3 })],
      output: null,
    },
    {
      code: inFunction("f(\n  alpha,\n  beta,\n  gamma,\n  delta,\n);"),
      options,
      errors: [
        {
          ...report(27, four),
          line: 2,
          column: 1,
          endLine: 7,
          endColumn: 1,
        },
      ],
      output: null,
    },
    {
      code: inFunction(
        "createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });",
      ),
      options: [{ maxWidth: 60, maxReferences: 3 }],
      errors: [
        {
          ...report(75, ["config", "resolveHost", "env", "loadCerts", "paths"], {
            maxWidth: 60,
            maxReferences: 3,
          }),
          column: 12,
        },
      ],
      output: null,
    },
    {
      code: inFunction("const alpha = 1;\nf([alpha, beta], { gamma });"),
      options,
      errors: [report(26, ["alpha", "beta", "gamma"])],
      output: null,
    },
    {
      code: inFunction('f("a string long enough to be wide on its own", alpha, beta, gamma);'),
      options: [{ maxWidth: 23, maxReferences: 2 }],
      errors: [report(24, ["alpha", "beta", "gamma"], { maxWidth: 23, maxReferences: 2 })],
      output: null,
    },
    {
      code: inFunction("f(`a long template ${alpha} with text ${beta} around ${gamma}`);"),
      options,
      errors: [report(27, ["alpha", "beta", "gamma"])],
      output: null,
    },
    {
      code: inFunction("new Server(alpha, beta, gamma, delta);"),
      options,
      errors: [{ ...report(27, four), column: 10 }],
      output: null,
    },
    {
      code: inFunction("f?.(alpha, beta, gamma, delta);"),
      options,
      errors: [{ ...report(27, four), column: 3 }],
      output: null,
    },
    {
      code: inFunction("outer(inner(alpha, beta, gamma), delta);"),
      options: [{ maxWidth: 19, maxReferences: 2 }],
      errors: [
        {
          ...report(20, ["alpha", "beta", "gamma"], { maxWidth: 19, maxReferences: 2 }),
          column: 11,
        },
      ],
      output: null,
    },
    {
      code: inFunction(
        "outer(alpha, beta, gamma, (item) => {\n  inner(alpha, beta, gamma, delta);\n});",
      ),
      options,
      errors: [
        { ...report(34, ["alpha", "beta", "gamma"]), line: 2, column: 5 },
        { ...report(27, four), line: 3, column: 7 },
      ],
      output: null,
    },
    {
      code: inFunction("f((value = compute(alpha, beta)) => value);"),
      options,
      errors: [{ ...report(38, ["compute", "alpha", "beta"]), column: 1 }],
      output: null,
    },
    {
      code: inFunction("render(<Panel title={heading} open={visible}>{label}</Panel>);"),
      filename: "view.tsx",
      options,
      errors: [{ messageId: "tooAssembled" }],
      output: null,
    },
    {
      code: inFunction("f(alpha, beta, gamma, delta);"),
      filename: "input.js",
      options,
      errors: [{ messageId: "tooAssembled" }],
      output: null,
    },
    {
      code: "const run = () => f(alpha, beta, gamma, delta);",
      options,
      errors: [{ ...report(27, four), column: 19 }],
      output: null,
    },
    {
      code: "class Holder {\n  field = f(alpha, beta, gamma, delta);\n}",
      options,
      errors: [{ ...report(27, four), line: 2, column: 11 }],
      output: null,
    },
    {
      code: "function scope() {\n  useless(alpha, beta, gamma, delta);\n  user(alpha, beta, gamma, delta);\n}",
      options,
      errors: [
        { ...report(27, four), line: 2 },
        { ...report(27, four), line: 3 },
      ],
      output: null,
    },
    {
      code: "function scope() {\n  useMemo(() => combine(alpha, beta, gamma, delta), []);\n}",
      options,
      errors: [{ ...report(27, four), line: 2, column: 23 }],
      output: null,
    },
    {
      code: "export const handler = defineHandler(async () => {\n  await save(alpha, beta, gamma, delta);\n});",
      options,
      errors: [{ ...report(27, four), line: 2, column: 12 }],
      output: null,
    },
  ],
});
