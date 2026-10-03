import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

export function newTypedRule(root: string, name: string): void {
  if (!/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(name)) {
    throw new Error("Expected a kebab-case rule name.");
  }
  const ruleDir = resolve(root, "native/rules", name);
  const docsDir = resolve(root, "docs/typed-rules");
  const docPath = resolve(docsDir, `${name}.md`);
  const jsRuleDir = resolve(root, "src/rules", name);
  if (existsSync(ruleDir) || existsSync(docPath) || existsSync(jsRuleDir)) {
    throw new Error(`Rule or documentation already exists: ${name}`);
  }
  mkdirSync(ruleDir, { recursive: true });
  mkdirSync(docsDir, { recursive: true });
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
  writeFileSync(
    resolve(ruleDir, "rule_test.go"),
    `package ${goPackage}

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{{Code: "const value = 1;"}},
		[]rule_tester.InvalidTestCase{{Code: "const value = 2;", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "unexpected"}}}},
	)
}
`,
    { flag: "wx" },
  );
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
    docPath,
    `# ${name}

## Intent

TODO: describe the behavior and why existing rules do not cover it.

## Runtime

Native Go rule, run by Oxlint through the aexlint/typed-plugin JavaScript plugin as aexlint-typed/${name}.

## Type information

TODO: specify the exact type queries, handling of any/unknown/unions/generics, and unresolved imports.

## Reported code

TODO: examples and precise reporting locations.

## Accepted code

TODO: near misses and a cross-file example where only the imported type changes the result.

## Options

None. Validate options in Go before using them.

## Fixes

None. The typed plugin does not forward fixes; Go rule tests can verify fixes and suggestions.

## Limitations

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to tsc. No editor integration yet.
`,
    { flag: "wx" },
  );
}

if (import.meta.main) {
  const [name, ...extra] = process.argv.slice(2);
  try {
    if (!name || extra.length) throw new Error("Usage: pnpm rule:new:typed <kebab-case-name>");
    newTypedRule(process.cwd(), name);
    console.log(
      `Created native ${name}. Replace the intentionally failing examples, then run pnpm native:test ${name}.`,
    );
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
