import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { globSync, mkdtempSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { test } from "node:test";
import plugin from "../src/index.ts";

const root = fileURLToPath(new URL("../", import.meta.url));
const oxlint = fileURLToPath(new URL("../node_modules/oxlint/bin/oxlint", import.meta.url));

function run(command: string, args: string[], cwd: string, expectedStatus = 0): string {
  const result = spawnSync(command, args, { cwd, encoding: "utf8", timeout: 60_000 });
  if (result.error) throw result.error;
  assert.equal(
    result.status,
    expectedStatus,
    `${command} ${args.join(" ")}\n${result.stdout}\n${result.stderr}`,
  );
  return result.stdout;
}

test("packed package imports and executes in a real Oxlint consumer", { timeout: 120_000 }, (t) => {
  const consumer = mkdtempSync(join(tmpdir(), "aexlint-consumer-"));
  t.after(() => rmSync(consumer, { recursive: true, force: true }));
  const packed = JSON.parse(
    run(
      "pnpm",
      ["--config.ignore-scripts=true", "pack", "--json", "--pack-destination", consumer],
      root,
    ),
  );
  const { filename, files } = packed as { filename: string; files: { path: string }[] };
  assert.ok(files.some((file) => file.path === "dist/index.js"));
  assert.ok(files.some((file) => file.path === "dist/index.d.ts"));
  const target = `${process.platform}-${process.arch}`;
  const executable = `aexlint-typed${process.platform === "win32" ? ".exe" : ""}`;
  assert.ok(files.some((file) => file.path === `dist/native/${target}/${executable}`));
  assert.ok(files.some((file) => file.path === "dist/native/THIRD_PARTY_NOTICES.txt"));
  assert.ok(files.every((file) => !/probe|\.go$/.test(file.path)));
  assert.ok(
    files.every((file) => !/\.test\.|^(?:src|tests|scripts|node_modules)\//.test(file.path)),
  );
  for (const { path } of files) {
    assert.match(
      path,
      /^(?:dist\/|docs\/(?:typed-)?rules\/[^/]+\.md$|(?:README\.md|LICENSE|package\.json)$)/,
    );
    assert.doesNotMatch(path, /(?:^|\/)\.[^/]+|\.map$/);
  }
  const packedDocs = files.map((file) => file.path).filter((path) => path.startsWith("docs/"));
  const ruleDocs = globSync(["docs/rules/*.md", "docs/typed-rules/*.md"], { cwd: root });
  assert.deepEqual(packedDocs.toSorted(), ruleDocs.toSorted());
  for (const path of ["README.md", "LICENSE", "package.json"]) {
    assert.ok(
      files.some((file) => file.path === path),
      `Missing ${path}`,
    );
  }
  writeFileSync(join(consumer, "package.json"), JSON.stringify({ private: true, type: "module" }));
  const store = run("pnpm", ["store", "path"], root).trim();
  run(
    "pnpm",
    ["add", "--lockfile-only", "--ignore-scripts", "--store-dir", store, filename],
    consumer,
  );
  run("pnpm", ["fetch", "--frozen-lockfile", "--ignore-scripts", "--store-dir", store], consumer);
  run(
    "pnpm",
    ["install", "--offline", "--frozen-lockfile", "--ignore-scripts", "--store-dir", store],
    consumer,
  );
  const imported = run(
    process.execPath,
    [
      "--input-type=module",
      "-e",
      'import plugin from "@aetherall/aexlint"; console.log(JSON.stringify({name: plugin.meta.name, rules: Object.keys(plugin.rules).toSorted()}));',
    ],
    consumer,
  );
  assert.deepEqual(JSON.parse(imported), {
    name: "aexlint",
    rules: Object.keys(plugin.rules).toSorted(),
  });

  writeFileSync(
    join(consumer, "consumer.ts"),
    'import plugin from "@aetherall/aexlint";\nimport typedPlugin from "@aetherall/aexlint/typed-plugin";\nexport const rules = plugin.rules;\nexport const typedRules = typedPlugin.rules;\n',
  );
  const compiler = fileURLToPath(
    new URL("./bin/tsc", import.meta.resolve("typescript/package.json")),
  );
  run(
    process.execPath,
    [compiler, "--noEmit", "--strict", "--module", "NodeNext", "--target", "ES2023", "consumer.ts"],
    consumer,
  );

  writeFileSync(
    join(consumer, "probe.js"),
    `import plugin from "@aetherall/aexlint";
export default {
  ...plugin,
  rules: {
    ...plugin.rules,
    "package-probe": {
      meta: { schema: [], messages: { unexpected: "Package probe executed." } },
      create(context) {
        return { DebuggerStatement(node) { context.report({ node, messageId: "unexpected" }); } };
      },
    },
  },
};
`,
  );
  writeFileSync(
    join(consumer, ".oxlintrc.json"),
    JSON.stringify({
      categories: { correctness: "off" },
      jsPlugins: ["./probe.js"],
      rules: { "aexlint/package-probe": "error" },
    }),
  );
  writeFileSync(join(consumer, "invalid.ts"), "debugger;\n");
  const diagnostics = JSON.parse(
    run(
      process.execPath,
      [oxlint, "-c", ".oxlintrc.json", "--format", "json", "invalid.ts"],
      consumer,
      1,
    ),
  );
  assert.equal(diagnostics.diagnostics.length, 1);
  assert.equal(diagnostics.diagnostics[0].message, "Package probe executed.");
  writeFileSync(join(consumer, "valid.ts"), "export const value: number = 1;\n");
  run(process.execPath, [oxlint, "-c", ".oxlintrc.json", "valid.ts"], consumer);

  writeFileSync(
    join(consumer, ".oxlintrc.json"),
    JSON.stringify({ jsPlugins: ["@aetherall/aexlint"], categories: { correctness: "off" } }),
  );
  run(process.execPath, [oxlint, "-c", ".oxlintrc.json", "valid.ts"], consumer);
  const depthRule = "aexlint/max-expression-depth";
  const syntaxConfig = (ruleName: string, setting: unknown) =>
    writeFileSync(
      join(consumer, ".oxlintrc.json"),
      JSON.stringify({
        jsPlugins: ["@aetherall/aexlint"],
        categories: { correctness: "off" },
        rules: { [ruleName]: setting },
      }),
    );
  syntaxConfig(depthRule, ["error", { max: 2 }]);
  writeFileSync(join(consumer, "depth.ts"), "publish(encode(parse(input)));\n");
  const depthArgs = [oxlint, "-c", ".oxlintrc.json", "depth.ts", "--format", "json"];
  const depthDiagnostics = JSON.parse(run(process.execPath, depthArgs, consumer, 1));
  assert.equal(depthDiagnostics.diagnostics.length, 1);
  assert.match(depthDiagnostics.diagnostics[0].message, /Input-production depth is 3/);
  writeFileSync(
    join(consumer, "depth.ts"),
    "const document = parse(input); publish(encode(document));\n",
  );
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  const complexityRule = "aexlint/max-expression-complexity";
  writeFileSync(join(consumer, "depth.ts"), "result(a(), b(), c(), d());\n");
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  syntaxConfig(complexityRule, ["error", { max: 4 }]);
  const workload = JSON.parse(run(process.execPath, depthArgs, consumer, 1));
  assert.equal(workload.diagnostics.length, 1);
  assert.match(workload.diagnostics[0].message, /Expression workload is 5/);
  writeFileSync(join(consumer, "depth.ts"), "const first = a(); result(first, b(), c(), d());\n");
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  writeFileSync(
    join(consumer, "depth.ts"),
    "const value = { one: a(), two: b(), three: c(), four: d(), five: e() };\nconst all = a && b && c && d && e;\n",
  );
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  const decisionRule = "aexlint/max-decision-depth";
  const alternatives = "const value = (a && b) || (c && d);\n";
  const nested = "const value = a && (b || (c && d));\n";
  for (const code of [alternatives, nested]) {
    writeFileSync(join(consumer, "depth.ts"), code);
    syntaxConfig(complexityRule, ["error", { max: 2 }]);
    const messages = JSON.parse(run(process.execPath, depthArgs, consumer, 1)).diagnostics;
    assert.equal(messages.length, 1);
    assert.match(messages[0].message, /Expression workload is 3/);
  }
  syntaxConfig(decisionRule, ["error", { max: 2 }]);
  const decisions = JSON.parse(run(process.execPath, depthArgs, consumer, 1)).diagnostics;
  assert.equal(decisions.length, 1);
  assert.match(decisions[0].message, /Decision depth is 3/);
  writeFileSync(join(consumer, "depth.ts"), alternatives);
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  const heldRule = "aexlint/max-held-context";
  writeFileSync(join(consumer, "depth.ts"), "if (a) { if (b) { if (c) { if (d) work(); } } }\n");
  syntaxConfig(heldRule, ["error", { max: 3 }]);
  const held = JSON.parse(run(process.execPath, depthArgs, consumer, 1)).diagnostics;
  assert.equal(held.length, 1);
  assert.match(held[0].message, /held-context depth 4/);
  writeFileSync(
    join(consumer, "depth.ts"),
    "for (const x of xs) { if (!x) continue; if (x.a) { use(x); } }\n",
  );
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  syntaxConfig("aexlint/prefer-domain-predicate", "error");
  writeFileSync(
    join(consumer, "depth.ts"),
    "if (node.type === 'Array' || node.type === 'Object') use(node);\n",
  );
  const classification = JSON.parse(run(process.execPath, depthArgs, consumer, 1));
  assert.equal(classification.diagnostics.length, 1);
  assert.match(classification.diagnostics[0].message, /classification of node/);
  writeFileSync(
    join(consumer, "depth.ts"),
    "function isContainer(node) { return node.type === 'Array' || node.type === 'Object'; }\n",
  );
  assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  for (const code of [
    "const isX = a.amount > 10 && a.type === 'money';",
    "const types = ['Array', 'Object']; if (types.includes(node.type)) use(node);",
  ]) {
    writeFileSync(join(consumer, "depth.ts"), code);
    assert.deepEqual(JSON.parse(run(process.execPath, depthArgs, consumer)).diagnostics, []);
  }
  writeFileSync(
    join(consumer, "depth.ts"),
    "const types = ['Array', 'Object']; if (types.includes(node.type) && node.state === 'ready') use(node);",
  );
  const combined = JSON.parse(run(process.execPath, depthArgs, consumer, 1));
  assert.equal(combined.diagnostics.length, 1);
  assert.match(combined.diagnostics[0].message, /classification of node/);
  for (const ruleName of [depthRule, complexityRule, decisionRule, heldRule]) {
    for (const setting of [
      "error",
      ["error", {}],
      ["error", { max: 0 }],
      ["error", { max: 1.5 }],
      ["error", { max: "2" }],
      ["error", { max: Number.MAX_SAFE_INTEGER + 1 }],
      ["error", { max: 2, extra: true }],
      ["error", { max: 2 }, {}],
    ]) {
      syntaxConfig(ruleName, setting);
      const result = spawnSync(process.execPath, depthArgs, {
        cwd: consumer,
        encoding: "utf8",
        timeout: 30_000,
      });
      if (result.error) throw result.error;
      assert.notEqual(result.status, 0, JSON.stringify(setting));
      assert.match(result.stdout + result.stderr, /(?:config|option|schema|max)/i);
    }
  }
  const typedRules = JSON.parse(
    run(
      process.execPath,
      [
        "--input-type=module",
        "-e",
        'import plugin from "@aetherall/aexlint/typed-plugin"; console.log(JSON.stringify(Object.keys(plugin.rules)));',
      ],
      consumer,
    ),
  );
  assert.ok(typedRules.includes("prefer-truthy-presence-check"));
  assert.ok(typedRules.every((name: string) => !name.includes("probe")));
  writeFileSync(
    join(consumer, "tsconfig.json"),
    JSON.stringify({ compilerOptions: { strict: true, types: [] }, files: ["typed-input.ts"] }),
  );
  writeFileSync(
    join(consumer, ".oxlintrc.json"),
    JSON.stringify({
      jsPlugins: ["@aetherall/aexlint/typed-plugin"],
      categories: { correctness: "off" },
      rules: { "aexlint-typed/prefer-truthy-presence-check": "error" },
    }),
  );
  const caller = 'import { value } from "./dependency.js";\nif (value !== undefined) {}\n';
  writeFileSync(join(consumer, "typed-input.ts"), caller);
  writeFileSync(
    join(consumer, "dependency.ts"),
    "export declare const value: { id: string } | undefined;",
  );
  const typedArgs = [oxlint, "-c", ".oxlintrc.json", "--format", "json", "typed-input.ts"];
  const presence = JSON.parse(run(process.execPath, typedArgs, consumer, 1));
  assert.equal(presence.diagnostics.length, 1);
  assert.equal(presence.diagnostics[0].code, "aexlint-typed(prefer-truthy-presence-check)");
  assert.match(presence.diagnostics[0].message, /Use a truthiness check/);
  writeFileSync(join(consumer, "dependency.ts"), "export declare const value: string | undefined;");
  assert.deepEqual(JSON.parse(run(process.execPath, typedArgs, consumer)).diagnostics, []);
  assert.equal(readFileSync(join(consumer, "typed-input.ts"), "utf8"), caller);
});
