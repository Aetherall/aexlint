import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { test } from "node:test";
import { root, ruleNames } from "../scripts/native.ts";
import { differences, expectedProject, lintProject } from "../scripts/project.ts";
import { rules } from "../src/rules.ts";

test("every rule has project cases with expected findings", () => {
  const expected = expectedProject();
  const codes = [
    ...Object.keys(rules).map((name) => [name, `aexlint(${name})`]),
    ...ruleNames().map((name) => [name, `aexlint(${name})`]),
  ];
  for (const [name, code] of codes) {
    assert.ok(existsSync(join(root, "tests/project", name!)), `${name}: no tests/project folder`);
    assert.ok(expected.includes(` ${code}\n`), `${name}: no expected finding`);
  }
});

test("every rule reports exactly the expected findings on the test project", () => {
  assert.equal(differences(expectedProject(), lintProject()), "");
});
