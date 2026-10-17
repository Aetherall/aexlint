import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { test } from "node:test";
import plugin from "../src/index.ts";
import { rules, typedRuleNames } from "../src/rules.ts";

const rulesDir = new URL("../rules/", import.meta.url);

test("every rule folder has one implementation, a doc and a registry entry", () => {
  const names = readdirSync(rulesDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .toSorted();
  const has = (name: string, file: string) => existsSync(new URL(`${name}/${file}`, rulesDir));
  const syntax = names.filter((name) => has(name, "rule.ts"));
  const typed = names.filter((name) => has(name, "rule.go"));
  assert.deepEqual(
    names.filter((name) => syntax.includes(name) === typed.includes(name)),
    [],
    "each rule has exactly one of rule.ts or rule.go",
  );
  assert.equal(plugin.meta?.name, "aexlint");
  assert.deepEqual(Object.keys(rules), syntax);
  assert.deepEqual(typedRuleNames, typed);
  assert.deepEqual(Object.keys(plugin.rules).toSorted(), names);
  for (const name of syntax) {
    const rule = rules[name];
    assert.ok(rule?.meta?.type, `${name}: missing rule type`);
    assert.ok(rule.meta.docs?.description, `${name}: missing description`);
    assert.equal(
      rule.meta.docs.requiresTypeChecking,
      false,
      `${name}: custom typed rules are not supported by Oxlint`,
    );
    assert.ok(rule.meta.schema, `${name}: missing options schema`);
    assert.ok(Object.keys(rule.meta.messages ?? {}).length, `${name}: missing messages`);
  }
  for (const name of names) {
    assert.ok(has(name, "README.md"), `${name}: missing README.md`);
    for (const file of ["README.md", syntax.includes(name) ? "rule.ts" : "rule.go"]) {
      const text = readFileSync(new URL(`${name}/${file}`, rulesDir), "utf8");
      assert.doesNotMatch(text, /\bTODO\b/, `${name}: unfinished ${file}`);
    }
  }
});
