import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { test } from "node:test";
import plugin from "../src/index.ts";

const rulesDir = new URL("../src/rules/", import.meta.url);
const docsDir = new URL("../docs/rules/", import.meta.url);

test("every shipped rule is registered, documented and tested", () => {
  const names = readdirSync(rulesDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .toSorted();
  assert.equal(plugin.meta?.name, "aexlint");
  assert.deepEqual(Object.keys(plugin.rules).toSorted(), names);
  assert.deepEqual(
    readdirSync(docsDir)
      .filter((name) => name.endsWith(".md"))
      .toSorted(),
    names.map((name) => `${name}.md`),
  );
  for (const name of names) {
    const rule = plugin.rules[name];
    assert.ok(rule?.meta?.type, `${name}: missing rule type`);
    assert.ok(rule.meta.docs?.description, `${name}: missing description`);
    assert.equal(
      rule.meta.docs.requiresTypeChecking,
      false,
      `${name}: custom typed rules are not supported by Oxlint`,
    );
    assert.ok(rule.meta.schema, `${name}: missing options schema`);
    assert.ok(Object.keys(rule.meta.messages ?? {}).length, `${name}: missing messages`);
    const source = readFileSync(new URL(`${name}/index.ts`, rulesDir), "utf8");
    const tests = readFileSync(new URL(`${name}/${name}.test.ts`, rulesDir), "utf8");
    const docs = readFileSync(new URL(`${name}.md`, docsDir), "utf8");
    for (const [kind, text] of Object.entries({ source, tests, docs })) {
      assert.doesNotMatch(text, /\bTODO\b/, `${name}: unfinished ${kind}`);
    }
  }
});
