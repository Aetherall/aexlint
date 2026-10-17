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

export function newRule(root: string, name: string): void {
  const folder = ruleFolder(root, name);
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
  addProjectRule(root, name, `aexlint/${name}`);
  writeFileSync(
    resolve(folder, "README.md"),
    `# ${name}

TODO: the problem this rule finds and why existing Oxlint rules do not cover it.

\`\`\`json
{ "rules": { "aexlint/${name}": "warn" } }
\`\`\`

No options.

## Example

TODO: a reported example and an accepted near miss.

## Reporting and limits

TODO: message ID, span, and limitations. No fixes.
`,
    { flag: "wx" },
  );

  writeRegistry(root);
}

if (import.meta.main) {
  const [name, ...extra] = process.argv.slice(2);
  if (!name || extra.length > 0) {
    console.error("Usage: pnpm rule:new <kebab-case-name>");
    process.exitCode = 1;
  } else {
    try {
      newRule(process.cwd(), name);
      console.log(
        `Created ${name}. Define its contract and replace tests/project/${name}/ with real cases.`,
      );
      console.log("Run: pnpm test:project");
    } catch (error) {
      console.error(error instanceof Error ? error.message : String(error));
      process.exitCode = 1;
    }
  }
}
