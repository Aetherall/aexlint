package no_procedural_value_dispatch

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const order = `export type Delivery = "parcel" | "letter" | "email";
export interface Order { delivery: Delivery; ref: string }`

func procedural(code, snippet string, nth int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range nth {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	line := strings.Count(before, "\n") + 1
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "proceduralBranch", Line: line, Column: column, EndColumn: column + len(snippet)}
}

func files() map[string]string { return map[string]string{"order.ts": order} }

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	upload := `import type { Order } from "./order.js";
declare function weigh(path: string): Promise<string>;
declare function insure(path: string): Promise<string>;
declare const abroad: boolean;
export async function ship(orders: Order[]) {
  return Promise.all(orders.map(async (order) => {
    if (order.delivery === "parcel") {
      return { ...order, path: await weigh(order.ref) };
    }
    if (order.delivery === "letter" && !abroad) {
      try {
        return { ...order, path: await insure(order.ref) };
      } catch {}
    }
    return order;
  }));
}`
	switched := `import type { Order } from "./order.js";
declare function log(message: string): void;
export function describe(order: Order, verbose: boolean) {
  switch (order.delivery) {
    case "parcel":
      if (verbose) log(order.ref);
      return "parcel";
    case "letter":
      return "letter";
    default:
      return "other";
  }
}`
	elsed := `import type { Order } from "./order.js";
declare function log(message: string): void;
export function list(order: Order, paths: string[]) {
  if (order.delivery === "email") {
    for (const path of paths) log(path);
  } else {
    log(order.ref);
  }
}`
	ternary := `import type { Order } from "./order.js";
export const size = (order: Order, small: boolean) =>
  order.delivery === "parcel" ? (small ? 1 : 2) : order.delivery === "letter" ? 3 : 0;`
	callback := `import type { Order } from "./order.js";
declare function each(items: string[], run: (item: string) => void): void;
declare function log(message: string): void;
export function walk(order: Order, items: string[]) {
  switch (order.delivery) {
    case "parcel":
      return each(items, (item) => log(item.length > 3 ? item : order.ref));
    case "letter":
      return log(order.ref);
  }
}`
	narrowed := `type Shape = { kind: "circle"; radius: number } | { kind: "square"; side: number } | { kind: "line" };
declare function log(message: string): void;
export function draw(shape: Shape, steps: number[]) {
  if (shape.kind === "circle") return shape.radius;
  if (shape.kind === "square") {
    for (const step of steps) log(String(step * shape.side));
  }
  return 0;
}`
	closed := `import type { Kind } from "./kind.js";
declare function log(message: string): void;
export function f(kind: Kind, items: string[]) {
  switch (kind) {
    case "a":
      for (const item of items) log(item);
      return;
    case "b":
      return;
  }
}`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// Flat branches: values, JSX, single calls and straight sequences of effects.
			{Code: `import type { Order } from "./order.js";
declare function log(message: string): void;
declare function save(order: Order): void;
export function icon(order: Order) {
  switch (order.delivery) { case "parcel": return "box"; case "letter": return "envelope"; default: return "other"; }
}
export function store(order: Order) {
  if (order.delivery === "parcel") { log("parcel"); save(order); return; }
  else if (order.delivery === "letter") { save(order); }
}
export const View = ({ order }: { order: Order }) => order.delivery === "parcel" ? <b>{order.ref}</b> : order.delivery === "letter" ? <i>{order.ref}</i> : null;`, Tsx: true, Files: files()},
			// A single test without another branch is not a dispatch.
			{Code: `import type { Order } from "./order.js";
declare function log(message: string): void;
export function f(order: Order, paths: string[]) { if (order.delivery === "email") { for (const path of paths) log(path); } log(order.ref); }`, Files: files()},
			// Control flow in the default branch or in a nested function declaration does not count.
			{Code: `import type { Order } from "./order.js";
declare function log(message: string): void;
export function f(order: Order, paths: string[]) {
  switch (order.delivery) {
    case "parcel": {
      function each() { for (const path of paths) log(path); }
      return each;
    }
    case "letter":
      return null;
    default:
      for (const path of paths) log(path);
      return null;
  }
}`, Files: files()},
			// Tests of different subjects are not one dispatch, and broad types are not vocabularies.
			{Code: `import type { Order } from "./order.js";
declare function log(message: string): void;
export function f(a: Order, b: Order, name: string, paths: string[]) {
  if (a.delivery === "parcel") { for (const path of paths) log(path); }
  if (b.delivery === "letter") { log(b.ref); }
  switch (name) { case "x": if (paths.length) log(name); break; case "y": break; }
}`, Files: files()},
			// Cross-file: widening only the dependency's type removes the report.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = string;`}},
		},
		[]rule_tester.InvalidTestCase{
			// Sibling ifs whose branches await, try and test another condition.
			{Code: upload, Files: files(), Errors: []rule_tester.InvalidTestCaseError{procedural(upload, "order.delivery", 1)}},
			// A switch branch containing a decision.
			{Code: switched, Files: files(), Errors: []rule_tester.InvalidTestCaseError{procedural(switched, "order.delivery", 1)}},
			// An if/else on one value whose branch loops.
			{Code: elsed, Files: files(), Errors: []rule_tester.InvalidTestCaseError{procedural(elsed, "order.delivery", 1)}},
			// A ternary chain with a nested ternary.
			{Code: ternary, Files: files(), Errors: []rule_tester.InvalidTestCaseError{procedural(ternary, "order.delivery", 1)}},
			// A decision inside a callback written in the branch.
			{Code: callback, Files: files(), Errors: []rule_tester.InvalidTestCaseError{procedural(callback, "order.delivery", 1)}},
			// Narrowing the object by an earlier test does not split the dispatch.
			{Code: narrowed, Errors: []rule_tester.InvalidTestCaseError{procedural(narrowed, "shape.kind", 1)}},
			// Cross-file: the dependency's closed type makes the switch a dispatch.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = "a" | "b";`}, Errors: []rule_tester.InvalidTestCaseError{procedural(closed, "kind", 3)}},
		},
	)
}
