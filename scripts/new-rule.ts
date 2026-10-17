import { existsSync, mkdirSync, readdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { addProjectRule } from "./project.ts";

export function ruleFolder(root: string, name: string): string {
  if (!/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(name)) {
    throw new Error("Expected a kebab-case rule name, e.g. no-unsafe-example.");
  }
  const folder = resolve(root, "rules", name);
  if (existsSync(folder) || existsSync(resolve(root, "tests/project", name))) {
    throw new Error(`Rule already exists: ${name}`);
  }
  mkdirSync(folder, { recursive: true });
  return folder;
}

export function writeRegistry(root: string): void {
  const rulesDir = resolve(root, "rules");
  const names = readdirSync(rulesDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .toSorted();
  const syntax = names.filter((name) => existsSync(resolve(rulesDir, name, "rule.ts")));
  const typed = names.filter((name) => existsSync(resolve(rulesDir, name, "rule.go")));
  const imports = syntax.map(
    (name, index) => `import rule${index} from "../rules/${name}/rule.ts";`,
  );
  const entries = syntax.map((name, index) => `  "${name}": rule${index},`);
  const typedEntries = typed.map((name) => `  "${name}",`);
  const registry = [
    `import type { Rule } from "@oxlint/plugins";\n${imports.join("\n")}\n`,
    `export const rules: Record<string, Rule> = {\n${entries.join("\n")}\n};\n`,
    `export const typedRuleNames: string[] = [\n${typedEntries.join("\n")}\n];\n`,
  ];
  mkdirSync(resolve(root, "src"), { recursive: true });
  writeFileSync(resolve(root, "src/rules.ts"), registry.join("\n"));
}

function syntaxRule(folder: string, name: string): void {
  writeFileSync(
    resolve(folder, "rule.ts"),
    `import { defineRule } from "@oxlint/plugins";

export default defineRule({
  meta: {
    type: "suggestion",
    docs: { description: "TODO: describe the rule.", requiresTypeChecking: false },
    schema: [],
    messages: { unexpected: "TODO: explain the problem and what to do instead." },
  },
  create() {
    throw new Error("TODO: implement ${name} from its documented contract.");
  },
});
`,
    { flag: "wx" },
  );
}

function typedRule(folder: string, name: string): void {
  writeFileSync(
    resolve(folder, "rule.go"),
    `package ${name.replaceAll("-", "_")}

import "github.com/typescript-eslint/tsgolint/internal/rule"

var Rule = rule.Rule{
\tName: "aexlint/${name}",
\tRun: func(ctx rule.RuleContext, options any) rule.RuleListeners {
\t\tpanic("TODO: implement ${name} from its documented contract")
\t},
}
`,
    { flag: "wx" },
  );
  const metadata = {
    name: `aexlint/${name}`,
    description: "TODO: describe the rule.",
    requiresTypeChecking: true,
  };
  writeFileSync(resolve(folder, "rule.json"), JSON.stringify(metadata, null, 2) + "\n", {
    flag: "wx",
  });
}

export function newRule(root: string, name: string, typed = false): void {
  const folder = ruleFolder(root, name);
  if (typed) typedRule(folder, name);
  else syntaxRule(folder, name);
  addProjectRule(root, name, `aexlint/${name}`);
  writeFileSync(
    resolve(folder, "README.md"),
    `# ${name}

TODO: the problem this rule finds and the maintenance burden it causes.

\`\`\`json
{ "rules": { "aexlint/${name}": "warn" } }
\`\`\`

No options.

## Example

TODO: a reported example and an accepted near miss; for a type-aware rule, also one where only an imported type changes the result (cross-file).

## What counts

TODO: what the rule reads, including any/unknown/unions/generics and unresolved imports for a type-aware rule.

## Reporting and limits

TODO: message ID, span, and limitations. No fixes.
`,
    { flag: "wx" },
  );
  writeRegistry(root);
}

if (import.meta.main) {
  const args = process.argv.slice(2);
  const typed = args.includes("--typed");
  const [name, ...extra] = args.filter((arg) => arg !== "--typed");
  try {
    if (!name || extra.length) throw new Error("Usage: pnpm rule:new <kebab-case-name> [--typed]");
    newRule(process.cwd(), name, typed);
    console.log(`Created rules/${name}. Replace tests/project/${name}/ with real cases.`);
    console.log("Run: pnpm test:project");
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
