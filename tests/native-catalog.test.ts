import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, readdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { newTypedRule } from "../scripts/new-typed-rule.ts";
import { native as runNative, ruleNames } from "../scripts/native.ts";

const rulesDir = new URL("../rules/", import.meta.url);

test("native rules have metadata and query the model", () => {
  for (const name of ruleNames()) {
    const metadata = JSON.parse(readFileSync(new URL(`${name}/rule.json`, rulesDir), "utf8"));
    assert.equal(metadata.name, `aexlint/${name}`);
    assert.equal(metadata.requiresTypeChecking, true);
    assert.ok(metadata.description);
    assert.doesNotMatch(JSON.stringify(metadata), /\bTODO\b/, `Unfinished native rule: ${name}`);
    const source = readFileSync(new URL(`${name}/rule.go`, rulesDir), "utf8");
    assert.ok(source.includes(`"aexlint/${name}"`));
    for (const file of readdirSync(new URL(`${name}/`, rulesDir)).filter(
      (entry) => entry.endsWith(".go") && !entry.endsWith("_test.go"),
    )) {
      const code = readFileSync(new URL(`${name}/${file}`, rulesDir), "utf8");
      assert.doesNotMatch(
        code,
        /shim\/checker"|\bTypeChecker\.(?!TypeToString\b)/,
        `${name}/${file} reasons about types itself; query the model instead (docs/native.md)`,
      );
    }
  }
});

test("native commands reject unknown source actions", async () => {
  await assert.rejects(runNative("unknown"), /Unknown native action: unknown/);
});

test("native generator creates a checker-ready skeleton and refuses overwrites", (t) => {
  const root = mkdtempSync(join(tmpdir(), "aexlint-native-generator-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  assert.throws(() => newTypedRule(root, "../escape"), /kebab-case/);
  newTypedRule(root, "one-rule");
  const source = readFileSync(join(root, "rules/one-rule/rule.go"), "utf8");
  assert.match(source, /package one_rule/);
  assert.match(source, /panic\("TODO/);
  const config = JSON.parse(readFileSync(join(root, "tests/project/oxlint.json"), "utf8"));
  assert.deepEqual(config.overrides, [
    { files: ["one-rule/**"], rules: { "aexlint/one-rule": "error" } },
  ]);
  assert.match(readFileSync(join(root, "rules/one-rule/README.md"), "utf8"), /cross-file/);
  assert.match(readFileSync(join(root, "src/rules.ts"), "utf8"), /"one-rule",/);
  assert.throws(() => newTypedRule(root, "one-rule"), /already exists/);
  assert.equal(readFileSync(join(root, "rules/one-rule/rule.go"), "utf8"), source);
});
