import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { ruleFolder, writeRegistry } from "./new-rule.ts";
import { addProjectRule } from "./project.ts";

export function newTypedRule(root: string, name: string): void {
  const ruleDir = ruleFolder(root, name);
  const goPackage = name.replaceAll("-", "_");
  writeFileSync(
    resolve(ruleDir, "rule.go"),
    `package ${goPackage}

import "github.com/typescript-eslint/tsgolint/internal/rule"

var Rule = rule.Rule{
	Name: "aexlint/${name}",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		panic("TODO: implement ${name} from its documented contract")
	},
}
`,
    { flag: "wx" },
  );
  addProjectRule(root, name, `aexlint/${name}`);
  writeFileSync(
    resolve(ruleDir, "rule.json"),
    JSON.stringify(
      {
        name: `aexlint/${name}`,
        description: "TODO: describe the rule.",
        requiresTypeChecking: true,
      },
      null,
      2,
    ) + "\n",
    { flag: "wx" },
  );
  writeFileSync(
    resolve(ruleDir, "README.md"),
    `# ${name}

TODO: the problem this rule finds and the maintenance burden it causes.

\`\`\`json
{ "rules": { "aexlint/${name}": "warn" } }
\`\`\`

No options.

## Example

TODO: a reported example, and an accepted one where only an imported type changes the result (cross-file).

## What counts

TODO: the type queries, including any/unknown/unions/generics and unresolved imports.

## Reporting and limits

TODO: message ID, span, and limitations. No fixes.
`,
    { flag: "wx" },
  );
  writeRegistry(root);
}

if (import.meta.main) {
  const [name, ...extra] = process.argv.slice(2);
  try {
    if (!name || extra.length) throw new Error("Usage: pnpm rule:new:typed <kebab-case-name>");
    newTypedRule(process.cwd(), name);
    console.log(
      `Created native ${name}. Replace tests/project/${name}/ with real cases, then run pnpm test:project.`,
    );
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
