package no_repeated_value_group

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const channel = `export type Channel = "sms" | "push" | "email" | "mail";
export interface Notice { channel: Channel; to: string }
export enum Level { Low, Mid, High, Max }`

// at locates the nth occurrence of snippet in code as a rule tester error with one-based columns.
func at(id, code, snippet string, nth int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range nth {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	position := func(offset int) (int, int) {
		before := code[:offset]
		return strings.Count(before, "\n") + 1, offset - strings.LastIndex(before, "\n")
	}
	line, column := position(offset)
	endLine, endColumn := position(offset + len(snippet))
	return rule_tester.InvalidTestCaseError{MessageId: id, Line: line, Column: column, EndLine: endLine, EndColumn: endColumn}
}

func group(code, snippet string, nth int) rule_tester.InvalidTestCaseError {
	return at("repeatedValueGroup", code, snippet, nth)
}

func files(extra ...string) map[string]string {
	result := map[string]string{"channel.ts": channel}
	for i := 0; i+1 < len(extra); i += 2 {
		result[extra[i]] = extra[i+1]
	}
	return result
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	instant := `import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "sms" || n.channel === "push" ? n.to : ""; }`
	elsewhere := `import type { Notice } from "./channel.js";
export function deliver(n: Notice) { if (n.channel === "push" || n.channel === "sms") return n.to; return ""; }`
	mixed := `import type { Notice, Channel } from "./channel.js";
declare function load(types: Channel[]): void;
declare function pick<const T extends Channel[]>(types: T): T;
export function f(n: Notice, t: Channel) {
  if (t !== "email" && t !== "mail") load([]);
  switch (n.channel) {
    case "sms":
    case "push":
      return 1;
    default:
      return 2;
  }
}
load(["push", "sms"]);
pick(["sms", "push"]);`
	nullable := `import type { Notice, Channel } from "./channel.js";
export function f(n?: Notice) { if (n?.channel === "sms" || (n?.channel as Channel) === "push") return 1; return 0; }`
	enums := `import { Level } from "./channel.js";
const HIGH = [Level.High, Level.Max] as const;
export function f(l: Level) {
  if (HIGH.includes(l as typeof HIGH[number])) return 1;
  if (l === Level.Max || l === Level.High) return 2;
  return 0;
}`
	tsx := `import type { Notice } from "./channel.js";
export const F = (props: { n: Notice }) => <div>{(props.n.channel === "sms" || props.n.channel === "push") && <img />}</div>;`
	narrowed := `type Shape = { kind: "circle" } | { kind: "square" } | { kind: "line" } | { kind: "dot" };
export function f(shape: Shape) {
  if (shape.kind === "circle") return 0;
  if (shape.kind === "square" || shape.kind === "line") return 1;
  return [shape.kind === "line" || shape.kind === "square" ? 2 : 3];
}`
	closed := `import type { Kind } from "./kind.js";
export function f(k: Kind) { return k === "a" || k === "b" ? 1 : 0; }`
	other := `import type { Kind } from "./kind.js";
export function g(k: Kind) { if (k === "b" || k === "a") return 1; return 0; }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// A group written once is not repeated.
			{Code: instant, Files: files()},
			// Named predicates and named constants define a group; they are neither reported nor counted.
			{Code: `import type { Notice, Channel } from "./channel.js";
export function isInstant(n: Notice) { return n.channel === "sms" || n.channel === "push"; }
export const instant = (n: Notice) => n.channel === "push" || n.channel === "sms";
export const INSTANT: Channel[] = ["sms", "push"];
export class Rules { static instant = (n: Notice) => ["sms", "push"].includes(n.channel); }`, Files: files("other.ts", instant)},
			// Single values and all-but-one tests partition off one value, not a group.
			{Code: `import type { Notice } from "./channel.js";
export function f(n: Notice) { if (n.channel === "sms" || n.channel !== "email") return 1; return n.channel !== "email" ? 2 : 3; }`, Files: files("other.ts", `import type { Notice } from "./channel.js";
export function h(n: Notice) { return n.channel === "sms" || n.channel !== "email" ? 1 : 0; }`)},
			// Different groups of the same vocabulary are different.
			{Code: instant, Files: files("other.ts", `import type { Notice } from "./channel.js";
export function loud(n: Notice) { return n.channel === "sms" || n.channel === "email" ? 1 : 0; }`)},
			// Broad, unknown, any and unresolved types are not vocabularies.
			{Code: `export function f(s: string, u: unknown, x: any, m: Missing) {
  if (s === "sms" || s === "push") return 1;
  if (u === "sms" || u === "push") return 2;
  if (x === "sms" || x === "push") return 3;
  if (m === "sms" || m === "push") return 4;
  return s === "sms" || s === "push" ? 5 : 6;
}`, Files: files()},
			// Cross-file: widening only the dependency's type removes the report.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = string;`, "a.ts": other}},
			// A guarded JSX element counts its group once.
			{Code: tsx, Tsx: true, Files: files()},
		},
		[]rule_tester.InvalidTestCase{
			// The same group in another file, written in another order.
			{Code: instant, Files: files("other.ts", elsewhere), Errors: []rule_tester.InvalidTestCaseError{group(instant, `n.channel === "sms" || n.channel === "push"`, 1)}},
			// Forms: complement, case group, list argument and generic list argument, all in one file.
			{Code: mixed, Files: files(), Errors: []rule_tester.InvalidTestCaseError{
				group(mixed, `t !== "email" && t !== "mail"`, 1),
				group(mixed, "case \"sms\":\n    case \"push\"", 1),
				group(mixed, `["push", "sms"]`, 1),
				group(mixed, `["sms", "push"]`, 1),
			}},
			// The help names an existing predicate; nullable and asserted subjects use their declared type.
			{Code: nullable, Files: files("named.ts", `import type { Notice } from "./channel.js";
export function isInstant(n: Notice) { return n.channel === "sms" || n.channel === "push"; }`, "other.ts", elsewhere),
				Errors: []rule_tester.InvalidTestCaseError{group(nullable, `n?.channel === "sms" || (n?.channel as Channel) === "push"`, 1)}},
			// Enum members, and includes on a named constant list.
			{Code: enums, Files: files(), Errors: []rule_tester.InvalidTestCaseError{
				group(enums, `HIGH.includes(l as typeof HIGH[number])`, 1),
				group(enums, `l === Level.Max || l === Level.High`, 1),
			}},
			// TSX guard.
			{Code: tsx, Tsx: true, Files: files("other.ts", elsewhere), Errors: []rule_tester.InvalidTestCaseError{group(tsx, `props.n.channel === "sms" || props.n.channel === "push"`, 1)}},
			// Narrowing the object by an earlier test keeps the property's declared vocabulary.
			{Code: narrowed, Errors: []rule_tester.InvalidTestCaseError{
				group(narrowed, `shape.kind === "square" || shape.kind === "line"`, 1),
				group(narrowed, `shape.kind === "line" || shape.kind === "square"`, 1),
			}},
			// Cross-file: the dependency's closed type makes the tests a group.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = "a" | "b" | "c" | "d";`, "a.ts": other},
				Errors: []rule_tester.InvalidTestCaseError{group(closed, `k === "a" || k === "b"`, 1)}},
		},
	)
}

func TestOptions(t *testing.T) {
	for _, options := range []any{nil, map[string]any{}} {
		ParseOptions("no-repeated-value-group", options)
	}
	for _, options := range []any{[]any{}, true, map[string]any{"max": float64(2)}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("ParseOptions(%v) accepted invalid options", options)
				}
			}()
			ParseOptions("no-repeated-value-group", options)
		}()
	}
}
