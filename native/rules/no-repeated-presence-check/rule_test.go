package no_repeated_presence_check

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const items = `export type Item = { id: string; items: string[] };
declare function log(value: unknown): void;
export function values(value: Item | undefined): string[] {
  if (value === undefined) return [];
  return value.items;
}
export function label(value: Item | undefined): string {
  if (!value) return "—";
  return value.id;
}
export function create(value: Item | undefined): Item {
  if (!value) return { id: "new", items: [] };
  return value;
}
export function logged(value: Item | undefined): string[] {
  log(value);
  if (!value) return [];
  return value.items;
}
export function nonEmpty(value: Item | undefined): string[] {
  if (!value || value.items.length === 0) return [];
  return value.items;
}
export function pair(id: string, value: Item | undefined) {
  if (!value) return undefined;
  return id + value.id;
}
export function only(value: Item | null | undefined): string[] {
  if (value === undefined) return [];
  return value ? value.items : [];
}
export function upper(text: string | undefined) {
  if (text === undefined) return undefined;
  return text.toUpperCase();
}
export function withDefault(value: Item | undefined = { id: "", items: [] }): string[] {
  if (!value) return [];
  return value.items;
}
export function many(...list: (Item | undefined)[]): string[] {
  if (!list[0]) return [];
  return list[0].items;
}
export function show(value: unknown) {
  if (!value) return undefined;
  return String(value);
}
export function guarded(value: Item | undefined, on: boolean): string[] {
  return value && on ? value.items : [];
}
export function visit(value: Item | undefined) {
  if (value) {
    log(value.id);
  }
}
export const ready = (value: Item | undefined, on: boolean) => value !== undefined && on && value.items.length > 0;
export function bound(this: void, value: Item | undefined): string[] {
  if (!value) return [];
  return value.items;
}
export async function load(value: Item | undefined) {
  if (!value) return undefined;
  return value.items;
}
export function* each(value: Item | undefined) {
  if (!value) return undefined;
  yield* value.items;
}
export function fixed(value: Item | undefined): string[] {
  if (!value) return [];
  return ["fixed"];
}
export function required(value: Item): string[] {
  if (!value) return [];
  return value.items;
}
export const firstItem = (value: Item | undefined) => value && value.items[0];
export function touch(value: Item | undefined) {
  value && log(value.id);
}
export declare function external(value: Item | undefined): string[];
export declare function wrap(list: string[]): string[];
export declare function now(): number;
export declare function stamped(value: Item | undefined, at: number): string[];`

func repeated(code, snippet string, occurrence int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range occurrence {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "repeatedPresenceCheck", Line: strings.Count(before, "\n") + 1, Column: column, EndColumn: column + len(snippet)}
}

func invalid(code string, files map[string]string, snippet string, occurrence int) rule_tester.InvalidTestCase {
	return rule_tester.InvalidTestCase{Code: code, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(code, snippet, occurrence)}}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"items.ts": items}
	holder := func(value string) map[string]string {
		return map[string]string{"items.ts": items, "holder.ts": `import type { Item } from "./items.js"; export type Holder = { value: ` + value + ` };`}
	}
	header := `import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Item } from "./items.js";
`
	ternary := header + `export const f = (property: Item | undefined) => property ? values(property) : [];`
	negated := header + `export const f = (property: Item | undefined) => property === undefined ? [] : values(property);`
	alias := `import { values as read } from "./items.js";
import type { Item as Entry } from "./items.js";
export const f = (property: Entry | undefined) => property !== undefined ? read(property) : [];`
	method := `type Filter = { kind: string };
type Section = { filter?: Filter };
export class Parser {
  parseFilter(c: Filter | undefined) {
    if (!c) {
      return undefined;
    }
    return c.kind;
  }
  section(section: Section) {
    return section.filter ? this.parseFilter(section.filter) : undefined;
  }
}`
	logical := `type Coords = { lat: number };
function toShape(c?: Coords) {
  if (!c) return undefined;
  return { at: c.lat };
}
export const f = (coords: Coords | undefined) => coords && toShape(coords);`
	generic := `export const first = <T,>(list: T[] | undefined) => (list ? list[0] : undefined);
export const f = (list?: string[]) => (list ? first(list) : undefined);`
	predicate := `type Text = { type: "text" };
type Image = { type: "image" };
type Leaflet = Text | Image;
function isText(m: Leaflet | undefined): m is Text {
  return m !== undefined && m.type === "text";
}
export const f = (m: Leaflet | undefined) => (m !== undefined ? isText(m) : false);`
	either := header + `export const f = (property: Item | undefined) => property ? nonEmpty(property) : [];`
	bare := `type User = { name: string };
function name(u?: User) {
  if (u == null) return;
  return u.name;
}
export const f = (user: User | undefined) => (user != null ? name(user) : undefined);`
	unknown := header + `export const f = (value: unknown) => (value ? show(value) : undefined);`
	property := `import { only } from "./items.js";
import type { Holder } from "./holder.js";
export const f = (h: Holder) => (h.value ? only(h.value) : []);`
	wrapped := `import { guarded, visit, ready, bound, load, each, fixed, required, firstItem, touch, type Item } from "./items.js";
`
	conjunction := wrapped + `export const f = (property: Item | undefined, on: boolean) => (property ? guarded(property, on) : []);`
	statement := wrapped + `export const f = (property: Item | undefined) => (property ? visit(property) : undefined);`
	chain := wrapped + `export const f = (property: Item | undefined, on: boolean) => (property !== undefined ? ready(property, on) : false);`
	receiver := wrapped + `export const f = (property: Item | undefined) => (property ? bound(property) : []);`
	truthy := wrapped + `export const f = (property: Item | undefined) => (property ? firstItem(property) : undefined);`
	statementChain := wrapped + `export const f = (property: Item | undefined) => (property ? touch(property) : undefined);`
	cast := wrapped + `export const f = (property: Item | undefined) => (property ? required(property as any) : []);`
	jsx := header + `declare function List(props: { ids: string[] }): null;
export const View = ({ property }: { property: Item | undefined }) => <List ids={property ? values(property) : []} />;`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: header + `export const f = (property: Item | undefined) => values(property);`, Files: files},
			// A different fallback is the caller's own policy.
			{Code: header + `export const f = (property: Item | undefined) => property ? label(property) : "";`, Files: files},
			// A function that builds a default does something else with a missing value.
			{Code: header + `export const f = (property: Item | undefined) => property ? create(property) : { id: "new", items: [] };`, Files: files},
			// The rejection must be the function's first statement.
			{Code: header + `export const f = (property: Item | undefined) => property ? logged(property) : [];`, Files: files},
			// Other arguments must be safe to evaluate without the value.
			{Code: header + `export const f = (property: Item | undefined) => property ? pair(property.id, property) : undefined;`, Files: files},
			{Code: header + `export const f = (property: Item | undefined) => property ? stamped(property, now()) : [];`, Files: files},
			// The caller's truthiness check also removes "", which the function does not reject.
			{Code: header + `export const f = (text: string | undefined) => text ? upper(text) : undefined;`, Files: files},
			// The caller removes null, which the function does not reject.
			{Code: header + `export const f = (property: Item | null | undefined) => property != null ? only(property) : [];`, Files: files},
			// Default values and rest parameters replace or collect the argument.
			{Code: header + `export const f = (property: Item | undefined) => property ? withDefault(property) : [];`, Files: files},
			{Code: header + `export const f = (property: Item | undefined) => property ? many(property) : [];`, Files: files},
			// Async functions and generators wrap their results, so they never return the constant itself.
			{Code: wrapped + `export const f = (property: Item | undefined) => property ? load(property) : undefined;`, Files: files},
			{Code: wrapped + `export const f = (property: Item | undefined) => property ? each(property) : undefined;`, Files: files},
			// The parameter's type already rules out a missing value, so calling directly would not type-check.
			{Code: wrapped + `export const f = (property: Item | undefined) => property ? required(property) : [];`, Files: files},
			// A test whose other side does not use the value switches modes rather than rejecting it.
			{Code: wrapped + `export const f = (property: Item | undefined) => property ? fixed(property) : [];`, Files: files},
			// Declared functions have no body to read.
			{Code: header + `export const f = (property: Item | undefined) => property ? external(property) : [];`, Files: files},
			// The call must be the whole branch.
			{Code: header + `export const f = (property: Item | undefined) => property ? wrap(values(property)) : [];`, Files: files},
			// An early return covers more than the call.
			{Code: header + `export function f(property: Item | undefined) { if (!property) return []; return values(property); }`, Files: files},
			// The call runs when the value is missing, or receives another value.
			{Code: header + `export const f = (property: Item | undefined) => property ? [] : values(property);`, Files: files},
			{Code: header + `export const f = (a: Item | undefined, b: Item | undefined) => a ? values(b) : [];`, Files: files},
			// any can hold "" and 0, which values does not reject; never says nothing.
			{Code: header + `export const f = (property: any) => property ? values(property) : [];`, Files: files},
			{Code: header + `export const f = (property: never) => property ? values(property) : [];`, Files: files},
			// Unresolved functions are not followed.
			{Code: `import { values } from "./missing.js";
export const f = (property: { id: string } | undefined) => property ? values(property) : [];`},
			// Only the imported type differs from the reported property case.
			{Code: property, Files: holder("Item | null | undefined")},
		},
		[]rule_tester.InvalidTestCase{
			invalid(ternary, files, "property", 2),
			invalid(negated, files, "property === undefined", 1),
			invalid(alias, files, "property !== undefined", 1),
			invalid(method, nil, "section.filter", 1),
			invalid(logical, nil, "coords", 2),
			invalid(generic, nil, "list", 5),
			invalid(predicate, nil, "m !== undefined", 2),
			invalid(either, files, "property", 2),
			invalid(bare, nil, "user != null", 1),
			invalid(unknown, files, "value", 3),
			invalid(property, holder("Item | undefined"), "h.value", 1),
			{Code: jsx, Tsx: true, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(jsx, "property", 3)}},
			{Code: ternary, Files: files, Options: rule_tester.OptionsFromJSON[any](`{}`), Errors: []rule_tester.InvalidTestCaseError{repeated(ternary, "property", 2)}},
			invalid(conjunction, files, "property", 2),
			invalid(statement, files, "property", 2),
			invalid(chain, files, "property !== undefined", 1),
			invalid(receiver, files, "property", 2),
			invalid(truthy, files, "property", 2),
			invalid(statementChain, files, "property", 2),
			invalid(cast, files, "property", 2),
		},
	)
}
