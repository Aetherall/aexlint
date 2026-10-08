package max_interpretation_spread

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const kinds = `export type Kind = "a" | "b" | "c";
export interface Item { kind: Kind; label: string; enabled: boolean }
export interface File { type: "file" } export interface Folder { type: "folder" | "alias" } export type Entity = File | Folder;`

// consumers returns n files, each containing body with its own import of names from kinds.ts, plus kinds.ts itself.
func consumers(n int, names, body string) map[string]string {
	files := map[string]string{"kinds.ts": kinds}
	for i := range n {
		files[fmt.Sprintf("consumer%d.ts", i)] = fmt.Sprintf("import { %s } from \"./kinds.js\";\n%s\n", names, body)
	}
	return files
}

func with(files map[string]string, name, code string) map[string]string {
	files[name] = code
	return files
}

func limit(max int) any {
	return rule_tester.OptionsFromJSON[any](fmt.Sprintf(`{"max": %d}`, max))
}

func spread(line, column, endColumn int) []rule_tester.InvalidTestCaseError {
	return []rule_tester.InvalidTestCaseError{{MessageId: "spreadInterpretation", Line: line, Column: column, EndColumn: endColumn}}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	item := `import type { Item } from "./kinds.js";
export const f = (item: Item) => item.kind === "a";`
	equality := `export const g = (item: Item) => item.kind !== "b";`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// At the limit: the linted file and one other file.
			{Code: item, Options: limit(2), Files: consumers(1, "Item", equality)},
			// Many subset selections in one other file count once.
			{Code: item, Options: limit(2), Files: with(consumers(0, "", ""), "a.ts", `import type { Item } from "./kinds.js";
export function g(item: Item, other: Item) {
  if (item.kind === "b") return 1;
  switch (other.kind) { case "c": return 2; default: return 3; }
}`)},
			// Exhaustive switches without default, lookups, and comparisons without a single literal are not subset selections.
			{Code: item, Options: limit(1), Files: consumers(3, "Item, Kind", `const LABELS: Record<Kind, string> = { a: "A", b: "B", c: "C" };
export function g(item: Item, other: Item, text: string) {
  switch (item.kind) { case "a": return 1; case "b": return 2; case "c": return 3; }
  return [LABELS[item.kind], item.kind === other.kind, item.kind === text, item.label === "x", item.enabled === true, typeof item.kind === "string"];
}`)},
			// A file without a subset selection is not reported, however spread the vocabulary is elsewhere.
			{Code: `import type { Item } from "./kinds.js"; export const f = (item: Item) => item.kind;`, Options: limit(1), Files: consumers(3, "Item", equality)},
			// A different literal set is a different vocabulary, even when it contains this one.
			{Code: item, Options: limit(1), Files: consumers(2, "", `export const g = (kind: "a" | "b" | "c" | "d") => kind === "a";`)},
			// Single literals, booleans, and unions containing non-literals are not closed vocabularies.
			{Code: `export const f = (a: "a", b: boolean, c: "a" | string, d: "a" | 1 | object) => a === "a" || b === true || c === "a" || d === "a";`, Options: limit(1),
				Files: consumers(2, "", `export const g = (a: "a", c: "a" | string) => a === "a" || c === "a";`)},
			// Without options, the limit is 4.
			{Code: item, Files: consumers(3, "Item", equality)},
			// Cross-file: widening only the dependency's type removes the report.
			{Code: `import type { Kind } from "./kinds2.js"; export const f = (kind: Kind) => kind === "a";`, Options: limit(1), Files: map[string]string{
				"kinds2.ts": `export type Kind = string;`,
				"a.ts":      `import type { Kind } from "./kinds2.js"; export const g = (kind: Kind) => kind === "a";`,
			}},
		},
		[]rule_tester.InvalidTestCase{
			// Without options, five files exceed the default limit of 4.
			{Code: item, Files: consumers(4, "Item", equality), Errors: spread(2, 34, 43)},
			// Reported on the value of the linted file's first subset selection.
			{Code: item, Options: limit(2), Files: consumers(2, "Item", equality), Errors: spread(2, 34, 43)},
			// Forms: inequality, loose equality, a constant, a non-exhaustive switch, a switch with default, nullable and asserted values, TSX.
			{Code: item, Options: limit(5), Files: with(with(with(with(with(consumers(0, "", ""),
				"a.ts", `import type { Item } from "./kinds.js"; export const g = (item?: Item) => item?.kind != "b";`),
				"b.ts", `import type { Item } from "./kinds.js"; const C = "c"; export const g = (item: Item) => (item.kind as Item["kind"]) === C;`),
				"c.ts", `import type { Kind } from "./kinds.js"; export function g(kind: Kind) { switch (kind) { case "a": return 1; case "b": return 2; } }`),
				"d.ts", `import type { Kind } from "./kinds.js"; export function g(kind: Kind | undefined) { switch (kind!) { case "a": case "b": case "c": return 1; default: return 0; } }`),
				"e.tsx", `import type { Item } from "./kinds.js"; export const J = (props: { item: Item }) => <div>{props.item.kind === "c" ? "c" : ""}</div>;`),
				Errors: spread(2, 34, 43)},
			// Redeclared copies of the same literal set count together: a local, an inline parameter type, and a class member.
			{Code: `import type { Kind } from "./kinds.js";
export function f(kind: Kind) { const copy = kind; return copy === "a" || copy !== "b"; }`, Options: limit(2), Files: with(with(with(consumers(0, "", ""),
				"a.ts", `export function g(kind: "a" | "b" | "c") { switch (kind) { case "a": return 1; default: return 2; } }`),
				"b.ts", `export class Holder { constructor(readonly kind: "c" | "b" | "a") {} }`),
				"c.ts", `import { Holder } from "./b.js"; export const g = (holder: Holder) => holder.kind === "c";`),
				Errors: spread(2, 59, 63)},
			// The message names the linted file's own declaration, not the one other files read most.
			{Code: `import { Holder } from "./b.js"; export const f = (holder: Holder) => holder.kind === "c";`, Options: limit(1),
				Files: with(consumers(2, "Kind", `export const g = (kind: Kind) => kind === "a" || kind === "b";`),
					"b.ts", `export class Holder { constructor(readonly kind: "c" | "b" | "a") {} }`),
				Errors: spread(1, 71, 82)},
			// Discriminated unions, calls, generic constraints, enums, as-const objects, and numbers.
			{Code: `import type { Entity } from "./kinds.js"; export const f = (s: Entity) => s.type === "file";`, Options: limit(1),
				Files: consumers(1, "Entity", `export const g = (s: Entity) => s.type !== "alias";`), Errors: spread(1, 75, 81)},
			{Code: `import type { Kind } from "./kinds.js"; export const f = (mode: () => Kind) => mode() === "a";`, Options: limit(1),
				Files: consumers(1, "Kind", `export const g = <K extends Kind>(kind: K) => kind === "b";`), Errors: spread(1, 80, 86)},
			{Code: `import { Level } from "./level.js"; export const f = (l: Level) => l === Level.High;`, Options: limit(1), Files: map[string]string{
				"level.ts": `export enum Level { Low, High }`,
				"a.ts":     `import { Level } from "./level.js"; const HIGH = Level.High; export const g = (l: Level) => l !== HIGH;`,
			}, Errors: spread(1, 68, 69)},
			{Code: `export const Status = { Ready: "ready", Done: "done" } as const; export type Status = (typeof Status)[keyof typeof Status];
export const f = (s: Status) => s === "ready";`, Options: limit(1), Files: map[string]string{
				"a.ts": `import { Status } from "./file.js"; export function g(s: Status) { switch (s) { case Status.Ready: return 1; default: return 2; } }`,
			}, Errors: spread(2, 33, 34)},
			{Code: `export const f = (status: 1 | 2 | 3) => status === 1;`, Options: limit(1),
				Files: consumers(1, "", `export const g = (status: 3 | 2 | 1) => status !== 2;`), Errors: spread(1, 41, 47)},
			// Each vocabulary with subset selections in the file is reported once.
			{Code: `import type { Item, Entity } from "./kinds.js";
export const f = (item: Item, s: Entity) => item.kind === "a" || item.kind === "b" || s.type === "file";`, Options: limit(1),
				Files: consumers(1, "Item, Entity", `export const g = (item: Item, s: Entity) => item.kind === "c" && s.type === "folder";`),
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "spreadInterpretation", Line: 2, Column: 45, EndColumn: 54},
					{MessageId: "spreadInterpretation", Line: 2, Column: 87, EndColumn: 93},
				}},
			// Beyond five values, the message elides them and the help lists them all.
			{Code: `export type Big = "a" | "b" | "c" | "d" | "e" | "f"; export const f = (big: Big) => big === "f";`, Options: limit(1),
				Files: map[string]string{"a.ts": `import type { Big } from "./file.js"; export const g = (big: Big) => big !== "a";`}, Errors: spread(1, 85, 88)},
			// Cross-file: the dependency's closed type makes the comparisons count.
			{Code: `import type { Kind } from "./kinds2.js"; export const f = (kind: Kind) => kind === "a";`, Options: limit(1), Files: map[string]string{
				"kinds2.ts": `export type Kind = "a" | "b";`,
				"a.ts":      `import type { Kind } from "./kinds2.js"; export const g = (kind: Kind) => kind === "a";`,
			}, Errors: spread(1, 75, 79)},
		},
	)
}

func TestOptions(t *testing.T) {
	for options, want := range map[string]int{`{"max": 7}`: 7, `{}`: 4, `null`: 4} {
		if got := parseOptions(rule_tester.OptionsFromJSON[any](options)); got != want {
			t.Errorf("parseOptions(%s) returned %d, want %d", options, got, want)
		}
	}
	for _, options := range []any{[]any{}, true, map[string]any{"max": nil}, map[string]any{"max": float64(0)}, map[string]any{"max": 1.5},
		map[string]any{"max": "4"}, map[string]any{"max": float64(1 << 54)}, map[string]any{"max": float64(4), "min": float64(1)}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("parseOptions(%v) accepted invalid options", options)
				}
			}()
			parseOptions(options)
		}()
	}
}
