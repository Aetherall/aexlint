package no_overbroad_type_predicate

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

type predicateCase struct {
	rule_tester.InvalidTestCase
	messages []string
}

func rejected(code, comparison, target, subject, value string) predicateCase {
	offset := strings.Index(code, comparison)
	if offset < 0 {
		panic("Missing comparison: " + comparison)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return predicateCase{
		InvalidTestCase: rule_tester.InvalidTestCase{Code: code, Errors: []rule_tester.InvalidTestCaseError{{
			MessageId: "overbroadTypePredicate", Line: strings.Count(before, "\n") + 1,
			Column: column, EndLine: strings.Count(before, "\n") + 1, EndColumn: column + len(comparison),
		}}},
		messages: []string{fmt.Sprintf("This comparison rejects %s = %s, which the declared predicate target %q permits; a false result does not exclude that target.", subject, value, target)},
	}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	dependency := `import type { Product as ImportedProduct } from "./dependency.js";
type LocalProduct = ImportedProduct;
export function selected(value: LocalProduct | undefined): value is LocalProduct {
  return value?.category === "physical";
}`
	normalizingGetter := `class Live { kind: "a" | "b" = "b"; private brand = 0; get ready(): true { this.kind = "a"; return true; } }
type Dead = { kind: "a" | "b"; ready: false };

`
	dataReady := strings.Replace(normalizingGetter, `get ready(): true { this.kind = "a"; return true; }`, `ready: true = true;`, 1)
	mixedKind := `class Read { private brand = 0; get kind(): "a" | "b" { return "a"; } }
type Data = { kind: "a" | "b" };

`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule, []rule_tester.ValidTestCase{
		{Code: `type Product = { category: "physical" | "digital" };
function present(value: Product | undefined): value is Product { return value !== undefined; }`},
		{Code: `type Product = { category: string };
type Physical = Product & { category: "physical" };
function selected(value: Product | undefined): value is Physical { return value?.category === "physical"; }`},
		{Code: `type Item = { kind: "stock"; sku: string } | { kind: "text"; text: string };
function stock(value: Item): value is Extract<Item, { kind: "stock" }> { return value.kind === "stock"; }`},
		{Code: `type Item = { kind: "a" | "b" };
function policy(value: Item): boolean { return value.kind === "a"; }
declare const values: Item[];
const chosen = values.filter(policy);
const inferred = (value: Item | undefined) => value !== undefined;`, Options: map[string]any{}},
		{Code: `function validator(value: unknown): value is number { return typeof value === "number" && value > 0; }
function anyInput(value: any): value is { kind: "a" | "b" } { return value.kind === "a"; }
function unknownInput(value: unknown): value is "a" | "b" { return value === "a"; }
function anyTarget(value: "a" | "b"): value is any { return value === "a"; }
function unknownTarget(value: unknown): value is unknown { return value === "a"; }
function impossible(value: never): value is never { return value === "a"; }
function broken(value: Missing | undefined): value is Missing { return value?.kind === "a"; }`},
		{Code: `type Item = { kind: any } | { kind: unknown };
function selected(value: Item): value is Item { return value.kind === "a"; }`},
		{Code: `function generic<T extends { kind: "a" | "b" }>(value: T): value is T { return value.kind === "a"; }
function property<T>(value: { kind: T }): value is { kind: T } { return value.kind === "a"; }`},
		{Code: `type Item = { kind: "a" | "b" };
function union(value: Item): value is Item { return value.kind === "a" || value.kind === "b"; }
declare function normalize(value: Item): boolean;
function effect(value: Item): value is Item { return normalize(value) && value.kind === "a"; }
function assign(value: Item): value is Item { return (value.kind = "a") === "a" && value.kind === "a"; }`},
		{Code: `class Product { get category(): string { return "physical"; } }
function selected(value: Product): value is Product { return value.category === "physical"; }`},
		{Code: normalizingGetter + `function guard(v: Live | Dead): v is Live { return v.ready && v.kind === "a"; }`},
		{Code: normalizingGetter + `function guard(v: Live | Dead | undefined): v is Live { return !!v?.ready && v.kind === "a"; }`},
		{Code: normalizingGetter + `type Mixed = Live & { ready: true };
function guard(v: Mixed): v is Mixed { return v.ready && v.kind === "a"; }`},
		{Code: mixedKind + `type Mixed = Read | Data;
function guard(v: Mixed): v is Mixed { return v.kind === "a"; }`},
		{Code: mixedKind + `type Mixed = Read | Data;
function guard(v: Mixed | undefined): v is Mixed { return v?.kind === "a"; }`},
		{Code: mixedKind + `type Mixed = Read & Data;
function guard(v: Mixed): v is Mixed { return v.kind === "a"; }`},
		{Code: mixedKind + `type Mixed = Read & Data;
function guard(v: Mixed | undefined): v is Mixed { return v?.kind === "a"; }`},
		{Code: `type Item = { kind: "a" | "b" };
function cast(value: Item): value is Item { return (value.kind as "a") === "a"; }
function asserted(value: Item): asserts value is Item { if (value.kind !== "a") throw new Error(); }
function ladder(value: Item | undefined): value is Item { if (!value) return false; return value.kind === "a"; }`},
		{Code: `type Item = { kind: "a" | "b" };
function defaulted(value: Item, ignored = (value.kind = "a")): value is Item { return value.kind === "a"; }
function destructured(value: Item, { [value.kind = "a"]: ignored }: { a?: unknown }): value is Item { return value.kind === "a"; }
function harmlessDefault(value: Item, ignored = 1): value is Item { return value.kind === "a"; }
function harmlessBinding(value: Item, { ignored }: { ignored: number }): value is Item { return value.kind === "a"; }`},
		{Code: `type Item = { kind: "a" | "b" };
function nullable(value: Item | undefined): value is Item { return value.kind === "a"; }
function disjoint(value: Item): value is Item { return value.kind === 1; }
function element(value: Item): value is Item { return value["kind"] === "a"; }`},
		{Code: `type Item = { code: 1 | 2 };
declare const chosen: 1 | 2;
function selected(value: Item): value is Item { return value.code === chosen; }`},
		{Code: dependency, Files: map[string]string{"dependency.ts": `export type Product = { category: "physical" };`}},
		{Code: dependency, Files: map[string]string{"dependency.ts": `export type Product = { category: unknown };`}},
		{Code: dependency, Files: map[string]string{"dependency.ts": `export type Product = { category: any };`}},
		{Code: dependency, Files: map[string]string{"dependency.ts": `export type Product = { category: never };`}},
		{Code: dependency},
	}, nil)

	cases := []predicateCase{
		rejected(`type Product = { category: "physical" | "digital" };
function selected(value: Product | undefined): value is Product { return value?.category === "physical"; }`, `value?.category === "physical"`, "Product", "value?.category", `"digital"`),
		rejected(`type Product = { category: string; format: string };
function selected(value: Product | undefined): value is Product { return value?.category === "physical" && value.format === "boxed"; }`, `value?.category === "physical"`, "Product", "value?.category", `""`),
		rejected(`type Product = { category: "physical" | "digital" };
function selected(value: Product | null | undefined): value is Product { return value != null && value.category === "physical"; }`, `value.category === "physical"`, "Product", "value.category", `"digital"`),
		rejected(`type Item = { kind: "a" | "b"; ready: boolean };
function selected(value: Item): value is Item { return value.ready && value.kind === "a"; }`, `value.kind === "a"`, "Item", "value.kind", `"b"`),
		rejected(`type Item = { kind: "a" | "b" };
declare function external(value: Item): boolean;
function selected(value: Item): value is Item { return value.kind === "a" && external(value); }`, `value.kind === "a"`, "Item", "value.kind", `"b"`),
		rejected(`type Item = { kind: "a" | "b" };
function selected(value: Item): value is Item { return "a" !== value.kind; }`, `"a" !== value.kind`, "Item", "value.kind", `"a"`),
		rejected(`type Item = { code: 1 | 2 };
const selected = (value: Item): value is Item => value.code === 1;`, `value.code === 1`, "Item", "value.code", "2"),
		rejected(`function selected(value: "a" | "b"): value is "a" | "b" { return value === "a"; }`, `value === "a"`, `"a" | "b"`, "value", `"b"`),
		rejected(`type Product = { category: string };
const helpers = { selected(value: Product): value is Product { return value.category === ""; } };`, `value.category === ""`, "Product", "value.category", `"other"`),
		rejected(`type Item = { kind: "a" | "b" };
function selected(value: Item): value is Item { return value.kind === "a"; }
declare const values: Item[];
const chosen = values.filter(selected);`, `value.kind === "a"`, "Item", "value.kind", `"b"`),
		rejected(`type Product<T> = { category: "physical" | "digital"; data: T };
function selected(value: Product<any> | undefined): value is Product<any> { return value?.category === "physical"; }`, `value?.category === "physical"`, "Product<any>", "value?.category", `"digital"`),
		rejected(`type Item = { kind: "a" | "b" };
function selected(unrelated: string, value: Item): value is Item { return value.kind === "a"; }`, `value.kind === "a"`, "Item", "value.kind", `"b"`),
		rejected(`type Item = { kind: "stock"; sku: string } | { kind: "text"; text: string };
function selected(value: Item): value is Extract<Item, { kind: "stock" }> { return value.kind === "stock" && value.sku === "alpha"; }`, `value.sku === "alpha"`, `{ kind: "stock"; sku: string; }`, "value.sku", `""`),
		rejected(`type Item = { code: 1 | 2 };
const chosen = 1;
function selected(value: Item): value is Item { return value.code === chosen; }`, `value.code === chosen`, "Item", "value.code", "2"),
		rejected(normalizingGetter+`function guard(v: Live | Dead): v is Live { return v.kind === "a" && v.ready; }`, `v.kind === "a"`, "Live", "v.kind", `"b"`),
		rejected(normalizingGetter+`function guard(v: Live | Dead | undefined): v is Live { return v?.kind === "a" && v.ready; }`, `v?.kind === "a"`, "Live", "v?.kind", `"b"`),
		rejected(dataReady+`function guard(v: Live | Dead): v is Live { return v.ready && v.kind === "a"; }`, `v.kind === "a"`, "Live", "v.kind", `"b"`),
		rejected(dataReady+`function guard(v: Live | Dead | undefined): v is Live { return !!v?.ready && v.kind === "a"; }`, `v.kind === "a"`, "Live", "v.kind", `"b"`),
		rejected(dataReady+`type Mixed = Live & { ready: true };
function guard(v: Mixed): v is Mixed { return v.ready && v.kind === "a"; }`, `v.kind === "a"`, "Mixed", "v.kind", `"b"`),
	}
	for _, category := range []string{`"physical" | "digital"`, `string`, `"physical" | "digital"`} {
		value := `"digital"`
		if category == "string" {
			value = `""`
		}
		c := rejected(dependency, `value?.category === "physical"`, "Product", "value?.category", value)
		c.Files = map[string]string{"dependency.ts": "export type Product = { category: " + category + " };"}
		cases = append(cases, c)
	}
	first := rejected(`type Item = { kind: "a" | "b" };
function first(value: Item): value is Item { return value.kind === "a"; }
const second = (other: Item): other is Item => other.kind !== "b";`, `value.kind === "a"`, "Item", "value.kind", `"b"`)
	second := rejected(first.Code, `other.kind !== "b"`, "Item", "other.kind", `"b"`)
	first.Errors = append(first.Errors, second.Errors...)
	first.messages = append(first.messages, second.messages...)
	cases = append(cases, first)

	for i, c := range cases {
		t.Run(fmt.Sprintf("product-%d", i), func(t *testing.T) {
			checked := rule.Rule{Name: Rule.Name, Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
				forward := ctx.ReportNode
				count := 0
				ctx.ReportNode = func(n *ast.Node, message rule.RuleMessage) {
					if count >= len(c.messages) || message.Description != c.messages[count] {
						t.Errorf("unexpected product message %d: %q", count, message.Description)
					}
					if message.Help != "A type predicate promises that false excludes its target too. Review the declared target or use a boolean policy; no automatic fix is offered." {
						t.Errorf("unexpected help: %q", message.Help)
					}
					count++
					forward(n, message)
				}
				return Rule.Run(ctx, options)
			}}
			rule_tester.RunRuleTester(root, "tsconfig.json", t, &checked, nil, []rule_tester.InvalidTestCase{c.InvalidTestCase})
		})
	}
}

func TestStrictNullChecks(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	code := `type Item = { kind: "a" | "b" };
function selected(value: Item): value is Item { return value.kind === "a"; }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule, nil, []rule_tester.InvalidTestCase{{
		Code: code, TSConfig: "loose.json",
		Files:  map[string]string{"loose.json": `{"compilerOptions":{"strict":true,"strictNullChecks":false,"types":[]},"include":["*.ts"]}`},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "strictNullChecksRequired", Line: 1, Column: 1}},
	}})
}

func TestOptions(t *testing.T) {
	for _, options := range []any{nil, map[string]any{}} {
		model.ParseOptions("no-overbroad-type-predicate", options)
	}
	for _, options := range []any{[]any{}, true, "invalid", map[string]any{"max": float64(2)}} {
		t.Run(fmt.Sprintf("%v", options), func(t *testing.T) {
			defer func() {
				if reason := recover(); reason != "aexlint/no-overbroad-type-predicate accepts no options; omit options or use an empty object" {
					t.Errorf("unexpected invalid-options result: %v", reason)
				}
			}()
			model.ParseOptions("no-overbroad-type-predicate", options)
		})
	}
}
