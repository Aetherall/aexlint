import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { newRule } from "../scripts/new-rule.ts";

function temporaryRoot(t: { after: (fn: () => void) => void }): string {
  const root = mkdtempSync(join(tmpdir(), "aexlint-generator-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  return root;
}

test("generates source, project cases, docs and a sorted registry", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "z-rule");
  newRule(root, "a-rule");
  const registry = readFileSync(join(root, "src/rules.ts"), "utf8");
  assert.ok(registry.indexOf('"a-rule"') < registry.indexOf('"z-rule"'));
  const config = JSON.parse(readFileSync(join(root, "tests/project/oxlint.json"), "utf8"));
  assert.deepEqual(config.overrides, [
    { files: ["z-rule/**"], rules: { "aexlint/z-rule": "error" } },
    { files: ["a-rule/**"], rules: { "aexlint/a-rule": "error" } },
  ]);
  for (const name of ["a-rule", "z-rule"]) {
    const source = readFileSync(join(root, `rules/${name}/rule.ts`), "utf8");
    assert.match(source, /throw new Error/);
    assert.match(source, /requiresTypeChecking: false/);
    assert.ok(existsSync(join(root, `tests/project/${name}/reported.ts`)));
    assert.match(readFileSync(join(root, `rules/${name}/README.md`), "utf8"), /## Example/);
  }
});

test("generates a type-aware rule with metadata, docs and a registry entry", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "one-rule", true);
  const source = readFileSync(join(root, "rules/one-rule/rule.go"), "utf8");
  assert.match(source, /package one_rule/);
  assert.match(source, /Name: "aexlint\/one-rule"/);
  const metadata = JSON.parse(readFileSync(join(root, "rules/one-rule/rule.json"), "utf8"));
  assert.equal(metadata.requiresTypeChecking, true);
  assert.ok(existsSync(join(root, "rules/one-rule/README.md")));
  assert.match(readFileSync(join(root, "src/rules.ts"), "utf8"), /typedRuleNames[^]*"one-rule",/);
  const config = JSON.parse(readFileSync(join(root, "tests/project/oxlint.json"), "utf8"));
  assert.deepEqual(config.overrides, [
    { files: ["one-rule/**"], rules: { "aexlint/one-rule": "error" } },
  ]);
});

test("refuses duplicate rules without changing their files", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "one-rule");
  const registry = readFileSync(join(root, "src/rules.ts"), "utf8");
  const source = readFileSync(join(root, "rules/one-rule/rule.ts"), "utf8");
  assert.throws(() => newRule(root, "one-rule"), /already exists/);
  assert.equal(readFileSync(join(root, "src/rules.ts"), "utf8"), registry);
  assert.equal(readFileSync(join(root, "rules/one-rule/rule.ts"), "utf8"), source);
});

test("rejects invalid names and path traversal before writing", (t) => {
  const root = temporaryRoot(t);
  for (const name of ["", "../escape", "Foo", "a/b", "a--b", "a-", "--type-aware"]) {
    assert.throws(() => newRule(root, name), /kebab-case/);
  }
  assert.deepEqual(readdirSync(root), []);
});

test("generated TypeScript compiles", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "unfinished-rule");
  writeFileSync(join(root, "package.json"), JSON.stringify({ private: true, type: "module" }));
  const modules = fileURLToPath(new URL("../node_modules", import.meta.url));
  symlinkSync(modules, join(root, "node_modules"), "dir");
  copyFileSync(new URL("../tsconfig.json", import.meta.url), join(root, "tsconfig.json"));
  const compiler = fileURLToPath(
    new URL("./bin/tsc", import.meta.resolve("typescript/package.json")),
  );
  const compiled = spawnSync(process.execPath, [compiler, "--noEmit"], {
    cwd: root,
    encoding: "utf8",
    timeout: 30_000,
  });
  if (compiled.error) throw compiled.error;
  assert.equal(compiled.status, 0, compiled.stdout + compiled.stderr);
});
