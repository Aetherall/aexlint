#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";
import { findingOf, formatBaseline, parseBaseline, projectPath } from "./baseline.ts";

const ignoreFile = ".aexlintignore";

interface Reported {
  filename: string;
  code: string;
  message: string;
}

const help = `Usage: aexlint check [options] [paths...]

Check the current directory when no paths are supplied.

  --typed                  Include stable type-aware rules (requires a TS project)
  --experimental           Include experimental and prototype rules
  -f, --format <format>     Oxlint output format (default: default)
  --ignore-pattern <glob>   Additional ignore pattern; repeatable
  --baseline <file>        Skip diagnostics recorded in a baseline file
  --write-baseline <file>  Record current diagnostics in a baseline file instead of reporting them
  -h, --help                Show help
  -V, --version             Show version

Uses standalone defaults, not project Oxlint configurations.
Reads ignore globs from .aexlintignore in the current directory.
Baseline paths are relative to the current directory.
Typed experimental rules require both --typed and --experimental.
Use -- before paths beginning with a dash.
`;

function readIgnorePatterns(): string[] {
  if (!existsSync(ignoreFile)) return [];
  const lines = readFileSync(ignoreFile, "utf8").split(/\r?\n/);
  return lines.map((line) => line.trim()).filter((line) => line && !line.startsWith("#"));
}

function writeBaseline(file: string, output: string, codes: Set<string>): number {
  const { diagnostics } = JSON.parse(output) as { diagnostics: Reported[] };
  const root = process.cwd();
  const recorded = diagnostics.filter(({ code }) => codes.has(code));
  const entries = recorded.map(({ filename, code, message }) => ({
    file: projectPath(root, resolve(filename)),
    code,
    message: findingOf(message),
  }));
  writeFileSync(file, formatBaseline(entries));
  console.log(`aexlint: recorded ${entries.length} diagnostics in ${file}`);
  return 0;
}

function check(): number {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      typed: { type: "boolean" },
      experimental: { type: "boolean" },
      format: { type: "string", short: "f" },
      "ignore-pattern": { type: "string", multiple: true },
      baseline: { type: "string" },
      "write-baseline": { type: "string" },
      help: { type: "boolean", short: "h" },
      version: { type: "boolean", short: "V" },
    },
  });
  if (values.help) {
    console.log(help);
    return 0;
  }
  if (values.version) {
    const manifest = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
    console.log(manifest.version);
    return 0;
  }
  const [command, ...paths] = positionals;
  if (command !== "check") throw new Error("Expected 'check'. Run aexlint --help for usage.");
  const { baseline, "write-baseline": record } = values;
  if (baseline && record) {
    throw new Error("--baseline cannot be combined with --write-baseline.");
  }
  if (record && values.format) {
    throw new Error("--write-baseline cannot be combined with --format.");
  }
  if (baseline) parseBaseline(readFileSync(baseline, "utf8"));

  const rules: Record<string, unknown> = {
    "aexlint/max-expression-depth": ["error", { max: 3 }],
    "aexlint/max-expression-complexity": ["error", { max: 4 }],
    "aexlint/max-decision-depth": ["error", { max: 2 }],
  };
  const jsPlugins = [fileURLToPath(new URL("./baseline-plugin.js", import.meta.url))];
  if (values.experimental) {
    Object.assign(rules, {
      "aexlint/max-call-assembly": ["error", { maxWidth: 60, maxReferences: 3 }],
      "aexlint/max-held-context": ["error", { max: 3 }],
      "aexlint/no-else-if": "error",
      "aexlint/no-multiline-condition": "error",
      "aexlint/prefer-domain-predicate": "error",
    });
  }
  if (values.typed) {
    jsPlugins.push(fileURLToPath(new URL("./baseline-typed-plugin.js", import.meta.url)));
    rules["aexlint-typed/prefer-truthy-presence-check"] = "error";
    if (values.experimental) {
      Object.assign(rules, {
        "aexlint-typed/max-interpretation-spread": ["error", { max: 4 }],
        "aexlint-typed/max-projection-spread": ["error", { max: 2 }],
        "aexlint-typed/require-interface-implementations": "error",
        "aexlint-typed/no-forgeable-validated-result": "error",
        "aexlint-typed/no-caller-enforced-invariant": "error",
      });
    }
  }
  const directory = mkdtempSync(join(tmpdir(), "aexlint-cli-"));
  try {
    const config = join(directory, ".oxlintrc.json");
    writeFileSync(config, JSON.stringify({ categories: { correctness: "off" }, jsPlugins, rules }));
    const oxlint = fileURLToPath(
      new URL("./bin/oxlint", import.meta.resolve("oxlint/package.json")),
    );
    const args = [oxlint, "--config", config, "--disable-nested-config"];
    const format = record ? "json" : values.format;
    if (format) args.push("--format", format);
    const patterns = [...readIgnorePatterns(), ...(values["ignore-pattern"] ?? [])];
    for (const pattern of patterns) args.push("--ignore-pattern", pattern);
    args.push("--", ...(paths.length ? paths : ["."]));
    const { AEXLINT_BASELINE: _, ...env } = process.env;
    if (baseline) env.AEXLINT_BASELINE = resolve(baseline);
    const result = spawnSync(process.execPath, args, {
      env,
      encoding: "utf8",
      maxBuffer: 1024 * 1024 * 1024,
      stdio: ["inherit", record ? "pipe" : "inherit", "inherit"],
    });
    if (result.error) throw result.error;
    if (!record) return result.status ?? 1;
    if (result.status !== 0 && result.status !== 1) return result.status ?? 1;
    const codes = new Set(Object.keys(rules).map((rule) => rule.replace(/\/(.*)$/, "($1)")));
    return writeBaseline(record, result.stdout, codes);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
}

try {
  process.exitCode = check();
} catch (error) {
  console.error(`aexlint: ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 2;
}
