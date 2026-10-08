import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { availableRules, decodeFrames } from "../src/typed/backend.ts";

const oxlint = fileURLToPath(new URL("../node_modules/oxlint/bin/oxlint", import.meta.url));
const plugin = fileURLToPath(new URL("../src/typed/plugin.ts", import.meta.url));
const target = `${process.platform}-${process.arch}`;
const backend = fileURLToPath(
  new URL(
    `../.native/probe/${target}/aexlint-typed${process.platform === "win32" ? ".exe" : ""}`,
    import.meta.url,
  ),
);

interface OxlintDiagnostic {
  message: string;
  code: string;
  severity: string;
  filename: string;
  labels: { span: { offset: number } }[];
}

interface Run {
  status: number | null;
  diagnostics: OxlintDiagnostic[];
  output: string;
}

function project(t: { after: (fn: () => void) => void }) {
  const root = mkdtempSync(join(tmpdir(), "aexlint-typed-test-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const write = (name: string, text: string) => writeFileSync(join(root, name), text);
  write("package.json", JSON.stringify({ private: true, type: "module" }));
  write(
    "tsconfig.json",
    JSON.stringify({
      compilerOptions: { strict: true, target: "ESNext", module: "NodeNext", types: [] },
      include: ["*.ts"],
    }),
  );
  write("input.ts", 'import { value } from "./dependency.js";\nvalue();\n');
  write("dependency.ts", 'export function value(): string { return "hello"; }\n');
  const configure = (rules: Record<string, unknown>) =>
    write(
      ".oxlintrc.json",
      JSON.stringify({ jsPlugins: [plugin], categories: { correctness: "off" }, rules }),
    );
  configure({ "aexlint-typed/type-probe": "error" });
  const lint = (files: string[], executable = backend): Run => {
    const result = spawnSync(
      process.execPath,
      [oxlint, "-c", ".oxlintrc.json", "--format", "json", ...files],
      {
        cwd: root,
        encoding: "utf8",
        timeout: 60_000,
        env: { ...process.env, AEXLINT_TYPED_BACKEND: executable },
      },
    );
    if (result.error) throw result.error;
    const output = `${result.stdout}\n${result.stderr}`;
    const parsed = result.stdout.startsWith("{") ? JSON.parse(result.stdout) : { diagnostics: [] };
    return { status: result.status, diagnostics: parsed.diagnostics, output };
  };
  return { root, write, configure, lint };
}

function directBackend(root: string, files: string[], rules: unknown[]): unknown[] {
  const input = JSON.stringify({
    version: 2,
    configs: [{ file_paths: files.map((file) => join(root, file)), rules }],
  });
  const result = spawnSync(backend, ["headless"], { cwd: root, input, maxBuffer: 64 << 20 });
  if (result.error) throw result.error;
  return decodeFrames(result.stdout).diagnostics;
}

function order(a: { filename: string; offset: number }, b: { filename: string; offset: number }) {
  return a.filename.localeCompare(b.filename) || a.offset - b.offset;
}

function later(message: string): boolean {
  return message.includes("stopped earlier in this run");
}

function stoppedLast(messages: string[]): string[] {
  return [...messages.filter((message) => !later(message)), ...messages.filter(later)];
}

function codes(run: Run): string[] {
  return run.diagnostics.map((item) => `${item.filename} ${item.code}`).toSorted();
}

test("native checker resolves imports and generics, not identifier-name heuristics", (t) => {
  const { root, write, configure, lint } = project(t);
  assert.ok(availableRules(backend).includes("aexlint/type-probe"));
  assert.deepEqual(lint(["input.ts"]).diagnostics, []);
  const unchangedCaller = readFileSync(join(root, "input.ts"), "utf8");
  write("dependency.ts", "export function value(): number { return 42; }\n");
  const failure = lint(["input.ts"]);
  assert.equal(failure.status, 1);
  assert.deepEqual(codes(failure), ["input.ts aexlint-typed(type-probe)"]);
  assert.equal(failure.diagnostics[0]?.message, "Type checker resolved a number result.");
  assert.equal(failure.diagnostics[0]?.severity, "error");
  assert.equal(readFileSync(join(root, "input.ts"), "utf8"), unchangedCaller);
  configure({ "aexlint-typed/type-probe": "warn" });
  const warning = lint(["input.ts"]);
  assert.equal(warning.status, 0);
  assert.equal(warning.diagnostics[0]?.severity, "warning");
  configure({ "aexlint-typed/type-probe": "off" });
  assert.deepEqual(lint(["input.ts"]).diagnostics, []);
  configure({ "aexlint-typed/type-probe": "error" });
  write(
    "input.ts",
    'function identity<T>(x: T): T { return x; } identity("hello"); identity(42);\n',
  );
  assert.equal(lint(["input.ts"]).diagnostics.length, 1);
});

test("plugin reports what the backend reports, at the same byte offsets", (t) => {
  const { root, write, configure, lint } = project(t);
  write("kinds.ts", 'export type Kind = "a" | "b" | "c";\n');
  write(
    "one.ts",
    'import type { Kind } from "./kinds.js";\nexport const f = (k: Kind) => k === "a";\n',
  );
  write(
    "three.ts",
    'import type { Kind } from "./kinds.js";\nconst label = "é";\nexport function h(k: Kind, entry: { id: string } | undefined) {\n  if (entry !== undefined) return label;\n  return k === "c";\n}\n',
  );
  const rules = [
    { name: "aexlint/max-interpretation-spread", options: { max: 1 } },
    { name: "aexlint/prefer-truthy-presence-check" },
  ];
  configure({
    "aexlint-typed/max-interpretation-spread": ["warn", { max: 1 }],
    "aexlint-typed/prefer-truthy-presence-check": "error",
  });
  const files = ["kinds.ts", "one.ts", "three.ts"];
  const expected = (
    directBackend(root, files, rules) as {
      rule: string;
      file_path: string;
      range: { pos: number };
      message: { description: string; help?: string };
    }[]
  ).map((item) => ({
    code: `aexlint-typed(${item.rule.replace("aexlint/", "")})`,
    filename: item.file_path.slice(root.length + 1),
    offset: item.range.pos,
    message: item.message.help
      ? `${item.message.description}\nhelp: ${item.message.help}`
      : item.message.description,
  }));
  assert.equal(expected.length, 3);
  const actual = lint(files).diagnostics.map((item) => ({
    code: item.code,
    filename: item.filename,
    offset: item.labels[0]!.span.offset,
    message: item.message,
  }));
  assert.deepEqual(actual.toSorted(order), expected.toSorted(order));
});

test("interpretation spread counts unlinted files and respects disable comments", (t) => {
  const { write, configure, lint } = project(t);
  write("kinds.ts", 'export type Kind = "a" | "b" | "c";\n');
  for (const name of ["input", "other"]) {
    write(
      `${name}.ts`,
      'import type { Kind } from "./kinds.js";\nexport const f = (kind: Kind) => kind === "a";\n',
    );
  }
  configure({ "aexlint-typed/max-interpretation-spread": ["warn", { max: 1 }] });
  const spread = lint(["input.ts"]);
  assert.deepEqual(codes(spread), ["input.ts aexlint-typed(max-interpretation-spread)"]);
  assert.equal(
    spread.diagnostics[0]!.message.split("\n")[1],
    "help: Checked in 2 files. Other files, nearest first: other.ts. Declared as: Kind (kinds.ts).",
  );
  write(
    "other.ts",
    'import type { Kind } from "./kinds.js";\n// oxlint-disable-next-line aexlint-typed/max-interpretation-spread\nexport const f = (kind: Kind) => kind === "a";\n',
  );
  assert.deepEqual(codes(lint(["input.ts", "other.ts"])), [
    "input.ts aexlint-typed(max-interpretation-spread)",
  ]);
  configure({ "aexlint-typed/max-interpretation-spread": ["warn", { max: 2 }] });
  assert.deepEqual(lint(["input.ts"]).diagnostics, []);
  configure({ "aexlint-typed/max-interpretation-spread": "warn" });
  assert.deepEqual(lint(["input.ts"]).diagnostics, []);
});

test("an invalid option stops the backend once, with its reason and no stack trace", (t) => {
  const { write, configure, lint } = project(t);
  write("other.ts", "export const other = 1;\n");
  configure({ "aexlint-typed/max-interpretation-spread": ["warn", { max: 0 }] });
  const invalid = lint(["input.ts", "other.ts"]);
  assert.equal(invalid.status, 1);
  const messages = stoppedLast(invalid.diagnostics.map((item) => item.message));
  assert.equal(messages.length, 2);
  assert.match(
    messages[0]!,
    /aexlint-typed: The typed backend stopped: aexlint\/max-interpretation-spread: "max" must be a positive safe integer$/,
  );
  assert.match(messages[1]!, /aexlint-typed: The typed backend stopped earlier in this run/);
  assert.doesNotMatch(invalid.output, /\n\s+at |EBADF/);
});

test("presence rule validates options and requires strict null checking", (t) => {
  const { write, configure, lint } = project(t);
  const name = "aexlint-typed/prefer-truthy-presence-check";
  write(
    "input.ts",
    "declare const value: { id: string } | undefined; if (value !== undefined) {}\n",
  );
  for (const options of [true, 1, "unexpected", [], { unexpected: true }]) {
    configure({ [name]: ["error", options] });
    const invalid = lint(["input.ts"]);
    assert.equal(invalid.status, 1);
    assert.match(invalid.output, /accepts no options/);
  }
  for (const options of [null, {}]) {
    configure({ [name]: ["error", options] });
    assert.match(lint(["input.ts"]).diagnostics[0]?.message ?? "", /Use a truthiness check/);
  }
  write(
    "tsconfig.json",
    JSON.stringify({ compilerOptions: { strict: true, strictNullChecks: false, types: [] } }),
  );
  const diagnostics = lint(["input.ts"]).diagnostics;
  assert.equal(diagnostics.length, 1);
  assert.match(diagnostics[0]!.message, /requires strictNullChecks/);
});

test("setup mistakes fail rather than silently passing", (t) => {
  const { root, write, configure, lint } = project(t);
  configure({ "aexlint-typed/unknown": "error" });
  const unknown = lint(["input.ts"]);
  assert.notEqual(unknown.status, 0);
  assert.match(unknown.output, /unknown/);
  configure({ "aexlint-typed/type-probe": "error" });
  mkdirSync(join(root, "loose"));
  write("loose/loose.ts", "export const loose = 1;\n");
  const outside = lint(["loose/loose.ts"]);
  assert.equal(outside.status, 1);
  assert.match(outside.output, /aexlint-typed: File .* is not included in a TypeScript project/);
  assert.doesNotMatch(outside.output, /\n\s+at /);
  rmSync(join(root, "tsconfig.json"));
  const missing = lint(["input.ts"]);
  assert.equal(missing.status, 1);
  assert.match(missing.output, /is not included in a TypeScript project/);
});

test("tsconfig errors are left to the compiler and the files are still linted", (t) => {
  const { write, lint } = project(t);
  write(
    "tsconfig.json",
    JSON.stringify({
      compilerOptions: { strict: true, notAnOption: true, types: [] },
      include: ["*.ts"],
    }),
  );
  write("dependency.ts", "export function value(): number { return 42; }\n");
  const run = lint(["input.ts"]);
  assert.equal(run.status, 1);
  assert.deepEqual(codes(run), ["input.ts aexlint-typed(type-probe)"]);
});

test("a backend server that exits at startup fails the run instead of hanging", (t) => {
  const { root, lint } = project(t);
  const fake = join(root, "fake");
  mkdirSync(fake);
  writeFileSync(join(fake, "rules.json"), JSON.stringify(["aexlint/type-probe"]));
  writeFileSync(
    join(fake, "aexlint-typed"),
    "#!/bin/sh\necho 'fake backend failed' >&2\nexit 3\n",
    {
      mode: 0o755,
    },
  );
  const started = performance.now();
  const run = lint(["input.ts", "dependency.ts"], join(fake, "aexlint-typed"));
  assert.equal(run.status, 1);
  const messages = stoppedLast(run.diagnostics.map((item) => item.message));
  assert.match(
    messages[0]!,
    /aexlint-typed: The typed backend did not start: fake backend failed$/,
  );
  assert.match(messages[1]!, /stopped earlier in this run, so this file was not linted/);
  assert.ok(performance.now() - started < 5_000);
});

test("TypeScript's own errors are left to the compiler", (t) => {
  const { write, lint } = project(t);
  write("input.ts", 'export const value: number = "not a number";\n');
  const run = lint(["input.ts"]);
  assert.equal(run.status, 0);
  assert.deepEqual(run.diagnostics, []);
});

test("native framing rejects truncated messages", () => {
  assert.deepEqual(decodeFrames(Buffer.alloc(0)), { diagnostics: [], programFiles: [] });
  assert.throws(() => decodeFrames(Buffer.alloc(3)), /Truncated native message header/);
  const header = Buffer.alloc(5);
  header.writeUInt32LE(50);
  assert.throws(() => decodeFrames(header), /Truncated native message payload/);
});
