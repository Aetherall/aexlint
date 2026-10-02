import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  copyFileSync,
  mkdirSync,
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

test("generates source, failing examples, docs and a sorted registry", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "z-rule");
  newRule(root, "a-rule");
  const registry = readFileSync(join(root, "src/rules/index.ts"), "utf8");
  assert.ok(registry.indexOf('"a-rule"') < registry.indexOf('"z-rule"'));
  for (const name of ["a-rule", "z-rule"]) {
    const source = readFileSync(join(root, `src/rules/${name}/index.ts`), "utf8");
    assert.match(source, /throw new Error/);
    assert.match(source, /requiresTypeChecking: false/);
    assert.match(
      readFileSync(join(root, `src/rules/${name}/${name}.test.ts`), "utf8"),
      /ruleTester.run/,
    );
    assert.match(readFileSync(join(root, `docs/rules/${name}.md`), "utf8"), /## Accepted code/);
  }
});

test("refuses duplicate rules without changing their files", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "one-rule");
  const registry = readFileSync(join(root, "src/rules/index.ts"), "utf8");
  const source = readFileSync(join(root, "src/rules/one-rule/index.ts"), "utf8");
  assert.throws(() => newRule(root, "one-rule"), /already exists/);
  assert.equal(readFileSync(join(root, "src/rules/index.ts"), "utf8"), registry);
  assert.equal(readFileSync(join(root, "src/rules/one-rule/index.ts"), "utf8"), source);
});

test("rejects invalid names and path traversal before writing", (t) => {
  const root = temporaryRoot(t);
  for (const name of ["", "../escape", "Foo", "a/b", "a--b", "a-", "--type-aware"]) {
    assert.throws(() => newRule(root, name), /kebab-case/);
  }
  assert.deepEqual(readdirSync(root), []);
});

test("generated TypeScript compiles and its real rule tests fail for unfinished behavior", (t) => {
  const root = temporaryRoot(t);
  newRule(root, "unfinished-rule");
  writeFileSync(join(root, "package.json"), JSON.stringify({ private: true, type: "module" }));
  symlinkSync(
    fileURLToPath(new URL("../node_modules", import.meta.url)),
    join(root, "node_modules"),
    "dir",
  );
  mkdirSync(join(root, "tests"));
  copyFileSync(new URL("./rule-tester.ts", import.meta.url), join(root, "tests/rule-tester.ts"));
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
  const env = { ...process.env };
  delete env.NODE_TEST_CONTEXT;
  const result = spawnSync(
    process.execPath,
    ["--test", "src/rules/unfinished-rule/unfinished-rule.test.ts"],
    { cwd: root, env, encoding: "utf8", timeout: 30_000 },
  );
  if (result.error) throw result.error;
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stdout + result.stderr, /TODO: implement unfinished-rule/);
});
