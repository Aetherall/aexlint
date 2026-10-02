package type_probe

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func TestTypeProbe(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: `import { value } from "./dependency.js"; value();`, Files: map[string]string{
				"dependency.ts": `export function value(): string { return "hello"; }`,
			}},
			{Code: `function value<T>(x: T): T { return x; } value("hello");`},
		},
		[]rule_tester.InvalidTestCase{
			{Code: `import { value } from "./dependency.js"; value();`, Files: map[string]string{
				"dependency.ts": `export function value(): number { return 42; }`,
			}, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "numberResult", Line: 1, Column: 42, EndColumn: 49}}},
			{Code: `function value<T>(x: T): T { return x; } value(42);`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "numberResult"}}},
		},
	)
}
