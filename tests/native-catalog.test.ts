import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, readdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { newTypedRule } from "../scripts/new-typed-rule.ts";
import { native as runNative, ruleNames } from "../scripts/native.ts";

const native = new URL("../native/rules/", import.meta.url);
const docs = new URL("../docs/typed-rules/", import.meta.url);

test("native rules have contracts, real tests, unique names, and no unfinished placeholders", () => {
  const names = ruleNames();
  assert.deepEqual(
    readdirSync(docs)
      .filter((name) => name.endsWith(".md"))
      .toSorted(),
    names.map((name) => `${name}.md`),
  );
  const syntaxNames = readdirSync(new URL("../src/rules/", import.meta.url));
  for (const name of names) {
    assert.ok(!syntaxNames.includes(name), `Duplicate JS/native rule name: ${name}`);
    const metadata = JSON.parse(readFileSync(new URL(`${name}/rule.json`, native), "utf8"));
    assert.equal(metadata.name, `aexlint/${name}`);
    assert.equal(metadata.requiresTypeChecking, true);
    assert.ok(metadata.description);
    const source = readFileSync(new URL(`${name}/rule.go`, native), "utf8");
    assert.ok(source.includes(`"aexlint/${name}"`));
    for (const text of [
      source,
      JSON.stringify(metadata),
      readFileSync(new URL(`${name}/rule_test.go`, native), "utf8"),
      readFileSync(new URL(`${name}.md`, docs), "utf8"),
    ]) {
      assert.doesNotMatch(text, /\bTODO\b/, `Unfinished native rule: ${name}`);
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
  const source = readFileSync(join(root, "native/rules/one-rule/rule.go"), "utf8");
  assert.match(source, /package one_rule/);
  assert.match(source, /panic\("TODO/);
  assert.match(
    readFileSync(join(root, "native/rules/one-rule/rule_test.go"), "utf8"),
    /RunRuleTester/,
  );
  assert.match(readFileSync(join(root, "docs/typed-rules/one-rule.md"), "utf8"), /cross-file/);
  assert.throws(() => newTypedRule(root, "one-rule"), /already exists/);
  assert.equal(readFileSync(join(root, "native/rules/one-rule/rule.go"), "utf8"), source);
});
