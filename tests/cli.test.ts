import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { test, type TestContext } from "node:test";
import { fileURLToPath } from "node:url";

const cli = fileURLToPath(new URL("../dist/cli.js", import.meta.url));
const { version } = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
const depthViolation = "publish(encode(parse(read(input))));\n";

interface Diagnostic {
  code: string;
  severity: string;
  filename: string;
  message: string;
}

function snapshot(root: string): Record<string, string> {
  const files: Record<string, string> = {};
  for (const entry of readdirSync(root, { recursive: true, withFileTypes: true })) {
    const path = join(entry.parentPath, entry.name);
    files[path] = entry.isDirectory() ? "directory" : readFileSync(path).toString("base64");
  }
  return files;
}

function consumer(t: TestContext) {
  const root = mkdtempSync(join(tmpdir(), "aexlint-cli-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const write = (name: string, source: string) => {
    const path = join(root, name);
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, source);
  };
  write("package.json", JSON.stringify({ private: true, type: "module" }));
  const run = (args: string[], expectedStatus = 0) => {
    const before = snapshot(root);
    const result = spawnSync(process.execPath, [cli, ...args], {
      cwd: root,
      encoding: "utf8",
      timeout: 60_000,
    });
    if (result.error) throw result.error;
    assert.deepEqual(
      snapshot(root),
      before,
      "CLI must not create, remove, or modify consumer files",
    );
    assert.equal(result.signal, null);
    assert.equal(
      result.status,
      expectedStatus,
      `${args.join(" ")}\n${result.stdout}\n${result.stderr}`,
    );
    return result.stdout + result.stderr;
  };
  const check = (args: string[] = [], expectedStatus = 0): Diagnostic[] => {
    const output = run(["check", "--format", "json", ...args], expectedStatus);
    const parsed = JSON.parse(output) as { diagnostics: Diagnostic[] };
    assert.ok(Array.isArray(parsed.diagnostics));
    for (const diagnostic of parsed.diagnostics) {
      assert.equal(diagnostic.severity, "error");
      assert.match(diagnostic.code, /^aexlint(?:-typed)?\(/);
    }
    return parsed.diagnostics;
  };
  return { write, run, check };
}

function codes(diagnostics: Diagnostic[]): string[] {
  return diagnostics.map(({ code }) => code).toSorted();
}

test("CLI defaults to the current directory and enforces stable limits as errors", (t) => {
  const { write, check } = consumer(t);
  write("depth.ts", depthViolation);
  write("complexity.ts", "result(a(), b(), c(), d());\n");
  write("decision.ts", "const value = a && (b || (c && d));\n");
  assert.deepEqual(codes(check([], 1)), [
    "aexlint(max-decision-depth)",
    "aexlint(max-expression-complexity)",
    "aexlint(max-expression-depth)",
  ]);
  assert.deepEqual(codes(check(["depth.ts", "decision.ts"], 1)), [
    "aexlint(max-decision-depth)",
    "aexlint(max-expression-depth)",
  ]);
  write("depth.ts", "publish(encode(parse(input)));\n");
  write("complexity.ts", "result(a(), b(), c());\n");
  write("decision.ts", "const value = a && (b || c);\n");
  assert.deepEqual(check(), []);
});

test("CLI ignores root and nested Oxlint configs and never enables builtin rules", (t) => {
  const { write, check } = consumer(t);
  const config = JSON.stringify({
    rules: { "no-debugger": "error", "aexlint/max-expression-depth": "off" },
    ignorePatterns: ["**/violation.ts"],
  });
  write(".oxlintrc.json", config);
  write("nested/.oxlintrc.json", config);
  write("clean.ts", "debugger;\nconst unused = 1;\nif (true) {}\n");
  write("nested/clean.ts", "debugger;\nconst unused = 1;\n");
  write("violation.ts", depthViolation);
  write("nested/violation.ts", depthViolation);
  assert.deepEqual(check(["clean.ts", "nested/clean.ts"]), []);
  const diagnostics = check([], 1);
  assert.deepEqual(codes(diagnostics), [
    "aexlint(max-expression-depth)",
    "aexlint(max-expression-depth)",
  ]);
  assert.deepEqual(diagnostics.map(({ filename }) => filename.replaceAll("\\", "/")).toSorted(), [
    "nested/violation.ts",
    "violation.ts",
  ]);
});

test("CLI enables every nonstable syntax rule only with --experimental", (t) => {
  const { write, check } = consumer(t);
  write("else.ts", "if (a) work(); else if (b) other();\n");
  write("held.ts", "if (a) { if (b) { if (c) { if (d) work(); } } }\n");
  write("multiline.ts", "if (a &&\n b) work();\n");
  write("predicate.ts", "if (node.type === 'Array' || node.type === 'Object') use(node);\n");
  assert.deepEqual(check(), []);
  assert.deepEqual(codes(check(["--experimental"], 1)), [
    "aexlint(max-held-context)",
    "aexlint(no-else-if)",
    "aexlint(no-multiline-condition)",
    "aexlint(prefer-domain-predicate)",
  ]);
  write("held.ts", "if (a) { if (b) { if (c) work(); } }\n");
  assert.deepEqual(check(["--experimental", "held.ts"]), []);
});

test("CLI resolves imported types and gates experimental typed rules on both flags", (t) => {
  const { write, check } = consumer(t);
  write(
    "tsconfig.json",
    JSON.stringify({
      compilerOptions: { strict: true, target: "ESNext", module: "NodeNext", types: [] },
      include: ["*.ts"],
    }),
  );
  write("dependency.ts", "export declare const value: { id: string } | undefined;\n");
  write("input.ts", 'import { value } from "./dependency.js";\nif (value !== undefined) {}\n');
  write("contract.ts", "export interface Clock { now(): number; }\n");
  assert.deepEqual(check(), []);
  assert.deepEqual(check(["--experimental"]), []);
  assert.deepEqual(codes(check(["--typed"], 1)), ["aexlint-typed(prefer-truthy-presence-check)"]);
  assert.deepEqual(codes(check(["--typed", "--experimental"], 1)), [
    "aexlint-typed(prefer-truthy-presence-check)",
    "aexlint-typed(require-interface-implementations)",
  ]);
  write("dependency.ts", "export declare const value: string | undefined;\n");
  assert.deepEqual(check(["--typed"]), []);
});

test("CLI respects ignore files, repeated ignore patterns, and disable comments", (t) => {
  const { write, check } = consumer(t);
  write(".gitignore", "git-ignored.ts\n");
  write(".eslintignore", "ignored.ts\n");
  for (const name of ["git-ignored.ts", "ignored.ts", "first.ts", "second.ts"]) {
    write(name, depthViolation);
  }
  write(
    "disabled.ts",
    `// oxlint-disable-next-line aexlint/max-expression-depth\n${depthViolation}`,
  );
  assert.deepEqual(codes(check([], 1)), [
    "aexlint(max-expression-depth)",
    "aexlint(max-expression-depth)",
  ]);
  assert.deepEqual(check(["--ignore-pattern", "first.ts", "--ignore-pattern", "second.ts"]), []);
});

test("CLI preserves visible failures for missing typed projects and malformed source", (t) => {
  const { write, run } = consumer(t);
  write("input.ts", "export const value = 1;\n");
  assert.match(
    run(["check", "--format", "default", "--typed", "input.ts"], 1),
    /aexlint-typed: File .*input\.ts is not included in a TypeScript project.*tsconfig\.json/,
  );
  write("broken.ts", "export const = ;\n");
  const report = JSON.parse(run(["check", "--format", "json", "broken.ts"], 1));
  assert.equal(report.diagnostics[0].filename, "broken.ts");
  assert.equal(report.diagnostics[0].message, "Unexpected token");
});

test("CLI supports help/version, rejects invalid arguments, and honors --", (t) => {
  const { write, run, check } = consumer(t);
  for (const flag of ["--help", "-h"]) {
    assert.match(run([flag]), /check/);
    assert.match(run(["check", flag]), /--typed/);
  }
  for (const flag of ["--version", "-V"]) {
    assert.ok(run([flag]).includes(version));
  }
  for (const args of [
    ["unknown-command"],
    ["--unknown"],
    ["check", "--unknown"],
    ["check", "--format"],
    ["check", "--ignore-pattern"],
  ]) {
    assert.match(run(args, 2), /unknown|unexpected|invalid|requires|missing|expected/i);
  }
  write("--typed.ts", depthViolation);
  assert.deepEqual(codes(check(["--", "--typed.ts"], 1)), ["aexlint(max-expression-depth)"]);
});
