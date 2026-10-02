import { existsSync, mkdirSync, readdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

export function newRule(root: string, name: string): void {
  if (!/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(name)) {
    throw new Error("Expected a kebab-case rule name, e.g. no-unsafe-example.");
  }

  const rulesDir = resolve(root, "src/rules");
  const ruleDir = resolve(rulesDir, name);
  const docsDir = resolve(root, "docs/rules");
  const docPath = resolve(docsDir, `${name}.md`);
  const typedRuleDir = resolve(root, "native/rules", name);
  if (existsSync(ruleDir) || existsSync(docPath) || existsSync(typedRuleDir)) {
    throw new Error(`Rule or documentation already exists: ${name}`);
  }

  mkdirSync(rulesDir, { recursive: true });
  mkdirSync(docsDir, { recursive: true });
  mkdirSync(ruleDir);
  writeFileSync(
    resolve(ruleDir, "index.ts"),
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
  writeFileSync(
    resolve(ruleDir, `${name}.test.ts`),
    `import { ruleTester } from "../../../tests/rule-tester.ts";
import rule from "./index.ts";

ruleTester.run("${name}", rule, {
  valid: ["const value = 1;"],
  invalid: [
    {
      code: "const value = 2;",
      errors: [{ messageId: "unexpected" }],
      output: null,
    },
  ],
});
`,
    { flag: "wx" },
  );
  writeFileSync(
    docPath,
    `# ${name}

## Intent

TODO: define the problem and why existing Oxlint rules do not cover it.

## Runtime

Oxlint JavaScript plugin; syntax/scope only. No TypeScript type information.

## Reported code

TODO: minimal invalid examples and the precise reporting location.

## Accepted code

TODO: valid examples, near misses, and false-positive boundaries.

## Options

None.

## Fixes

None. Only add a fix when preserving behavior can be demonstrated.

## Limitations

TODO: syntax variants, deliberate exclusions, and known trade-offs.
`,
    { flag: "wx" },
  );

  const names = readdirSync(rulesDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .toSorted();
  const imports = names.map(
    (ruleName, index) => `import rule${index} from "./${ruleName}/index.ts";`,
  );
  const entries = names.map((ruleName, index) => `  "${ruleName}": rule${index},`);
  writeFileSync(
    resolve(rulesDir, "index.ts"),
    `import type { Rule } from "@oxlint/plugins";\n${imports.join("\n")}\n\nexport const rules: Record<string, Rule> = {\n${entries.join("\n")}\n};\n`,
  );
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
        `Created ${name}. Define its contract and replace the intentionally failing examples.`,
      );
      console.log(`Run: pnpm test:rule src/rules/${name}/${name}.test.ts`);
    } catch (error) {
      console.error(error instanceof Error ? error.message : String(error));
      process.exitCode = 1;
    }
  }
}
