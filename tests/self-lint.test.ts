import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { rules } from "../src/rules/index.ts";

const repository = new URL("../", import.meta.url);
const syntaxConfig = JSON.parse(readFileSync(new URL(".oxlintrc.json", repository), "utf8"));
const typedConfig = JSON.parse(readFileSync(new URL(".oxlintrc.typed.json", repository), "utf8"));
const classificationConfig = JSON.parse(
  readFileSync(new URL(".oxlintrc.classifications.json", repository), "utf8"),
);

const input = `
declare function step(value?: unknown): unknown;
declare function combine(...values: unknown[]): unknown;
declare const a: boolean, b: boolean, c: boolean, d: boolean;
combine(step(), step(), step(), step());
step(step(step(step())));
export const nested = a && (b || (c && d));
declare const entry: { id: string } | undefined;
if (entry !== undefined) step(entry.id);
declare const node: { type: string };
if (node.type === 'Array' || node.type === 'Object') step(node);
`;

test("mandatory and experimental lint configurations cover every rule and TS directory", (t) => {
  const configured = { ...syntaxConfig.rules, ...classificationConfig.rules };
  const enabled = Object.keys(configured).toSorted();
  const expected = Object.keys(rules)
    .map((name) => `aexlint/${name}`)
    .toSorted();
  assert.deepEqual(enabled, expected);
  const root = mkdtempSync(join(tmpdir(), "aexlint-self-lint-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  writeFileSync(join(root, "package.json"), JSON.stringify({ type: "module" }));
  writeFileSync(
    join(root, "tsconfig.json"),
    JSON.stringify({
      compilerOptions: { strict: true, target: "ESNext", module: "NodeNext", types: [] },
      include: ["**/*.ts"],
    }),
  );
  for (const directory of ["src", "scripts", "tests"]) {
    mkdirSync(join(root, directory));
    writeFileSync(join(root, directory, "input.ts"), input);
  }
  const plugin = fileURLToPath(new URL(syntaxConfig.jsPlugins[0], repository));
  writeFileSync(
    join(root, ".oxlintrc.json"),
    JSON.stringify({ ...syntaxConfig, jsPlugins: [plugin] }),
  );
  const oxlint = new URL("./bin/oxlint", import.meta.resolve("oxlint/package.json"));
  const { NODE_TEST_CONTEXT: _, ...env } = process.env;
  const result = spawnSync(
    process.execPath,
    [fileURLToPath(oxlint), "--deny-warnings", "--format", "json", "src", "scripts", "tests"],
    { cwd: root, env, encoding: "utf8" },
  );
  assert.ifError(result.error);
  assert.equal(result.status, 1, result.stderr || result.stdout);
  const diagnostics = JSON.parse(result.stdout).diagnostics;
  assert.equal(diagnostics.length, 9, JSON.stringify(diagnostics, null, 2));
  for (const qualified of Object.keys(syntaxConfig.rules)) {
    const name = qualified.slice("aexlint/".length);
    const matching = diagnostics.filter(
      (item: { code: string }) => item.code === `aexlint(${name})`,
    );
    assert.equal(matching.length, 3, name);
  }

  writeFileSync(
    join(root, ".oxlintrc.json"),
    JSON.stringify({ ...classificationConfig, jsPlugins: [plugin] }),
  );
  const classifications = spawnSync(
    process.execPath,
    [fileURLToPath(oxlint), "--format", "json", "src", "scripts", "tests"],
    { cwd: root, env, encoding: "utf8" },
  );
  assert.ifError(classifications.error);
  assert.equal(classifications.status, 0, classifications.stderr || classifications.stdout);
  const findings = JSON.parse(classifications.stdout).diagnostics;
  assert.equal(findings.length, 3, JSON.stringify(findings, null, 2));
  for (const finding of findings) {
    assert.equal(finding.code, "aexlint(prefer-domain-predicate)");
    assert.equal(finding.severity, "warning");
  }

  const target = `${process.platform}-${process.arch}`;
  const executable = process.platform === "win32" ? "aexlint-typed.exe" : "aexlint-typed";
  const backend = fileURLToPath(new URL(`dist/native/${target}/${executable}`, repository));
  const typedPlugin = fileURLToPath(new URL(typedConfig.jsPlugins[0], repository));
  writeFileSync(
    join(root, ".oxlintrc.json"),
    JSON.stringify({ ...typedConfig, jsPlugins: [typedPlugin] }),
  );
  const typed = spawnSync(
    process.execPath,
    [fileURLToPath(oxlint), "--deny-warnings", "--format", "json", "src", "scripts", "tests"],
    { cwd: root, env: { ...env, AEXLINT_TYPED_BACKEND: backend }, encoding: "utf8" },
  );
  assert.ifError(typed.error);
  assert.equal(typed.status, 1, typed.stderr || typed.stdout);
  const typedDiagnostics = JSON.parse(typed.stdout).diagnostics;
  assert.equal(typedDiagnostics.length, 3, JSON.stringify(typedDiagnostics, null, 2));
  for (const diagnostic of typedDiagnostics) {
    assert.equal(diagnostic.code, "aexlint-typed(prefer-truthy-presence-check)");
    assert.equal(diagnostic.severity, "error");
  }
});
