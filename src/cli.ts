#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

const help = `Usage: aexlint check [options] [paths...]

Check the current directory when no paths are supplied.

  --typed                  Include stable type-aware rules (requires a TS project)
  --experimental           Include experimental and prototype rules
  -f, --format <format>     Oxlint output format (default: default)
  --ignore-pattern <glob>   Additional ignore pattern; repeatable
  -h, --help                Show help
  -V, --version             Show version

Uses standalone defaults, not project Oxlint configurations.
Typed experimental rules require both --typed and --experimental.
Use -- before paths beginning with a dash.
`;

function check(): number {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      typed: { type: "boolean" },
      experimental: { type: "boolean" },
      format: { type: "string", short: "f" },
      "ignore-pattern": { type: "string", multiple: true },
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

  const rules: Record<string, unknown> = {
    "aexlint/max-expression-depth": ["error", { max: 3 }],
    "aexlint/max-expression-complexity": ["error", { max: 4 }],
    "aexlint/max-decision-depth": ["error", { max: 2 }],
  };
  const jsPlugins = [fileURLToPath(new URL("./index.js", import.meta.url))];
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
    jsPlugins.push(fileURLToPath(new URL("./typed/plugin.js", import.meta.url)));
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
    if (values.format) args.push("--format", values.format);
    for (const pattern of values["ignore-pattern"] ?? []) args.push("--ignore-pattern", pattern);
    args.push("--", ...(paths.length ? paths : ["."]));
    const result = spawnSync(process.execPath, args, { stdio: "inherit" });
    if (result.error) throw result.error;
    return result.status ?? 1;
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
