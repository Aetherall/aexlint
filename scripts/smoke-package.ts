import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const directory = process.argv[2];
if (!directory) throw new Error("Usage: pnpm package:smoke <directory containing one .tgz>");
const tarballs = readdirSync(directory).filter((file) => file.endsWith(".tgz"));
if (tarballs.length !== 1) throw new Error(`Expected one tarball in ${directory}: ${tarballs}`);
const tarball = resolve(directory, tarballs[0]!);
const pnpm = process.env.npm_execpath;
if (!pnpm) throw new Error("Run through pnpm package:smoke so the consumer install uses pnpm.");
const pnpmCommand = /\.c?js$/.test(pnpm) ? [process.execPath, pnpm] : [pnpm];
const manifest = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
const consumer = mkdtempSync(join(tmpdir(), "aexlint-smoke-"));

function run(command: string[], expectedStatus: number): string {
  const [executable, ...args] = command;
  const result = spawnSync(executable!, args, {
    cwd: consumer,
    encoding: "utf8",
    timeout: 300_000,
  });
  if (result.error) throw result.error;
  const output = `${command.join(" ")}\n${result.stdout}\n${result.stderr}`;
  assert.equal(result.status, expectedStatus, output);
  return result.stdout;
}

try {
  writeFileSync(join(consumer, "package.json"), JSON.stringify({ private: true, type: "module" }));
  run([...pnpmCommand, "add", tarball, `oxlint@${manifest.dependencies.oxlint}`], 0);
  writeFileSync(
    join(consumer, "tsconfig.json"),
    JSON.stringify({ compilerOptions: { strict: true, types: [] }, files: ["input.ts"] }),
  );
  writeFileSync(
    join(consumer, ".oxlintrc.json"),
    JSON.stringify({
      jsPlugins: ["@aetherall/aexlint", "@aetherall/aexlint/typed-plugin"],
      categories: { correctness: "off" },
      rules: {
        "aexlint/max-expression-depth": ["error", { max: 2 }],
        "aexlint-typed/prefer-truthy-presence-check": "error",
      },
    }),
  );
  writeFileSync(
    join(consumer, "dependency.ts"),
    "export declare const value: { id: string } | undefined;\nexport declare function wrap<T>(input: T): T;\n",
  );
  writeFileSync(
    join(consumer, "input.ts"),
    'import { value, wrap } from "./dependency.js";\nif (value !== undefined) wrap(wrap(wrap(value)));\n',
  );
  const oxlint = join(consumer, "node_modules/oxlint/bin/oxlint");
  const report = run([process.execPath, oxlint, "--format", "json", "input.ts"], 1);
  const codes = JSON.parse(report).diagnostics.map((item: { code: string }) => item.code);
  assert.deepEqual(codes.toSorted(), [
    "aexlint(max-expression-depth)",
    "aexlint-typed(prefer-truthy-presence-check)",
  ]);
  const cliReport = run(
    [...pnpmCommand, "exec", "aexlint", "check", "--typed", "--format", "json", "input.ts"],
    1,
  );
  const cliCodes = JSON.parse(cliReport).diagnostics.map((item: { code: string }) => item.code);
  assert.deepEqual(cliCodes, ["aexlint-typed(prefer-truthy-presence-check)"]);
  console.log(`${tarballs[0]} runs both plugins on ${process.platform}-${process.arch}.`);
} finally {
  rmSync(consumer, { recursive: true, force: true, maxRetries: 5 });
}
