package prefer_truthy_presence_check

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
	valid := []rule_tester.ValidTestCase{}
	for _, valueType := range []string{
		"string", "number", "boolean", "bigint", "any", "unknown", "never", "undefined", "null",
		"string | undefined", "number | undefined", "boolean | undefined", "bigint | undefined",
		"0 | undefined", "-0 | undefined", "false | undefined", "'' | undefined", "0n | undefined",
		"{ id: string } | null | undefined", "{ id: string } | false | undefined", "null | undefined",
		"{} | undefined", "Object | undefined", "{ toString(): string } | undefined",
		"(number & { brand: true }) | undefined", "({ a: string } & { b: string }) | undefined",
		"`prefix${string}` | undefined", "HTMLAllCollection | undefined", "object | undefined",
	} {
		valid = append(valid, rule_tester.ValidTestCase{Code: "declare const value: " + valueType + "; if (value !== undefined) {}"})
	}
	for _, expression := range []string{
		"const result = value !== undefined;", "function f() { return value !== undefined; }",
		"const result = (value !== undefined) && value;", "const result = (value === undefined) || value;",
		"if ((value !== undefined) ?? true) {}", "if (Boolean(value !== undefined)) {}",
		"if ((value !== undefined) as boolean) {}", "let result; if (result = value !== undefined) {}",
		"if (value !== null) {}", "if (value !== void 0) {}", "if (value) {}",
	} {
		valid = append(valid, rule_tester.ValidTestCase{Code: "declare const value: { id: string } | undefined; " + expression})
	}
	valid = append(valid,
		rule_tester.ValidTestCase{Code: "function f<T>(value: T) { if (value !== undefined) {} }"},
		rule_tester.ValidTestCase{Code: "function f<T extends string | undefined>(value: T) { if (value !== undefined) {} }"},
		rule_tester.ValidTestCase{Code: "function f(undefined: number, value: number | undefined) { if (value !== undefined) {} }"},
		rule_tester.ValidTestCase{Code: "enum E { Zero, One }; declare const value: E | undefined; if (value !== undefined) {}"},
		rule_tester.ValidTestCase{Code: "import { value } from './dependency.js'; if (value !== undefined) {}", Files: map[string]string{"dependency.ts": "export declare const value: string | undefined;"}},
		rule_tester.ValidTestCase{Code: "import { value } from './missing.js'; if (value !== undefined) {}"},
		rule_tester.ValidTestCase{Code: "declare const values: { id: string }[]; if (values[0] !== undefined) {}"},
		rule_tester.ValidTestCase{Code: "declare const value: { id: string } | undefined; const jsx = <div>{value !== undefined && <span />}</div>;", Tsx: true},
	)
	invalid := []rule_tester.InvalidTestCase{}
	for _, valueType := range []string{
		"{ id: string }", "string[]", "readonly [number]", "Date", "symbol", "true", "'hello'", "42", "-1", "1n", "-1n",
		"{ id: string } | true | 'hello' | 42 | symbol",
	} {
		invalid = append(invalid, rule_tester.InvalidTestCase{
			Code:   "declare const value: " + valueType + " | undefined; if (value !== undefined) {}",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}},
		})
	}
	for _, condition := range []string{
		"if (value !== undefined) {}", "while (value !== undefined) { break; }", "do {} while (value !== undefined);",
		"for (; value !== undefined;) { break; }", "const result = value !== undefined ? 1 : 2;",
		"const result = !(value !== undefined);", "if (true && (value !== undefined || false)) {}",
		"if (((value) !== (undefined))) {}", "if (undefined !== value) {}", "if (value != undefined) {}",
		"if (value != undefined) { if (value !== undefined) {} }",
	} {
		invalid = append(invalid, rule_tester.InvalidTestCase{
			Code:   "declare const value: { id: string } | undefined; " + condition,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}},
		})
	}
	for _, condition := range []string{"if (value === undefined) {}", "if (undefined == value) {}", "const result = !(value == undefined);"} {
		invalid = append(invalid, rule_tester.InvalidTestCase{
			Code:   "declare const value: { id: string } | undefined; " + condition,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferFalsyCheck"}},
		})
	}
	invalid = append(invalid,
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | undefined;\nif (value !== undefined) {}", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck", Line: 2, Column: 5, EndLine: 2, EndColumn: 24}}},
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | null | undefined; if (value != undefined) {}", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | null | undefined; if (undefined == value) {}", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferFalsyCheck"}}},
		rule_tester.InvalidTestCase{Code: "function f<T extends { id: string } | undefined>(value: T) { if (value !== undefined) {} }", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "type Entry = { id: string }; declare const value: Entry | undefined; if (value !== undefined) {}", Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "import { value } from './dependency.js'; if (value !== undefined) {}", Files: map[string]string{"dependency.ts": "export declare const value: { id: string } | undefined;"}, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | undefined; const jsx = <div>{value !== undefined ? <span /> : null}</div>;", Tsx: true, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | undefined; if (value !== undefined) {}", Options: map[string]any{}, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferTruthyCheck"}}},
		rule_tester.InvalidTestCase{Code: "declare const value: { id: string } | undefined; if (value !== undefined) {}", TSConfig: "loose.json", Files: map[string]string{"loose.json": `{"compilerOptions":{"strictNullChecks":false,"types":[]},"include":["*.ts"]}`}, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "strictNullChecksRequired", Line: 1, Column: 1}}},
	)
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule, valid, invalid)
}
