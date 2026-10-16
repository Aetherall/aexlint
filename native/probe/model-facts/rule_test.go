package model_facts

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const items = `export type TextItem = { kind: "text"; text: string; note?: string };
export type StockItem = { kind: "stock"; sku: string; count: number };
export type Item = TextItem | StockItem;
export type Size = "S" | "M" | "L";
export declare function render(item: Item): string;
export declare function log(text: string): void;
export class Box {
  constructor(readonly size: Size, readonly label?: string) {}
  get small() { return this.size === "S"; }
}
export function isStock(item: Item): item is StockItem { return item.kind === "stock"; }`

var files = map[string]string{"items.ts": items}

func facts(kind string, count int) []rule_tester.InvalidTestCaseError {
	errors := make([]rule_tester.InvalidTestCaseError, count)
	for i := range errors {
		errors[i] = rule_tester.InvalidTestCaseError{MessageId: kind}
	}
	return errors
}

func only(kinds ...string) map[string]any {
	return map[string]any{"facts": kinds}
}

func TestModelFacts(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: `export function f(size: string) { return size === "S" ? 1 : 2; }`, Options: only("site", "decision")},
			{Code: `export function f(item: any) { return item.kind === "a" ? 1 : 2; }`, Options: only("site", "decision")},
			{Code: `export function f(item: Missing) { return item.kind === "a" ? 1 : 2; }`, Options: only("site", "decision")},
			{Code: `export function f(item: never) { return item; }`, Options: only("test")},
			{Code: `import type { Item } from "./items.js";
export function f(item: Item) { return item.kind === "text" ? item.text : ""; }`, Files: map[string]string{"items.ts": `export type Item = { kind: string; text: string };`}, Options: only("site", "decision", "test")},
		},
		[]rule_tester.InvalidTestCase{
			{Code: `import type { Size } from "./items.js";
export function f(size: Size) {
  if (size === "S" || size === "M") return 1;
  if (size !== "L" && ready()) return 2;
  return 3;
}
declare function ready(): boolean;
export const small: Size[] = ["S", "M"];
export function g(size: Size) {
  switch (size) {
    case "S":
    case "M":
      return "narrow";
    default:
      return "wide";
  }
}`, Files: files, Options: only("site"), Errors: facts("site", 4)},
			{Code: `import type { Item, Box } from "./items.js";
export function f(item: Item, box: Box, flag: string | undefined, count: number | undefined, value: unknown) {
  if (item.kind === "stock") return item.sku;
  if (["S", "M"].includes(box.size)) log("small");
  if (box.label) log(box.label);
  if (flag === undefined) return "";
  if (!count) return 0;
  if (Boolean(value)) return 1;
  return item.kind === "text" && item.note !== undefined ? item.note.length : 2;
}
declare function log(text: string): void;`, Files: files, Options: only("test"), Errors: facts("test", 8)},
			{Code: `type Shape = { kind: "circle"; radius: number } | { kind: "square"; side: number };
export function area(shape: Shape) {
  if (shape.kind === "circle") return shape.radius;
  return shape.kind === "square" ? shape.side : 0;
}`, Options: only("test"), Errors: facts("test", 2)},
			{Code: `import { type Item, render, log } from "./items.js";
declare function wrap(item: Item): Item;
export function outcomes(item: Item) {
  switch (item.kind) {
    case "text": return render(item);
    case "stock": return { sku: item.sku, count: item.count };
  }
}
export function more(item: Item) {
  if (item.kind === "text") return item;
  if (item.kind === "stock") return log("x");
}
export function rest(item: Item) {
  return item.kind === "text" ? "label" : item.kind === "stock" ? undefined : "none";
}
export function steps(item: Item) {
  switch (item.kind) {
    case "text":
      log("a");
      log("b");
      break;
    case "stock":
      for (const c of [1, 2]) log(String(c));
      return 1;
  }
}
export function other(item: Item) {
  return item.kind === "text" ? item.text + "!" : wrap(item);
}`, Files: files, Options: only("decision"), Errors: facts("decision", 5)},
			{Code: `import type { Item } from "./items.js";
enum Tone { Loud = "loud", Calm = "calm" }
declare const settings: { width: number };
export class View {
  scale = 2;
  constant(item: Item) { return item.kind === "text" ? { tone: Tone.Loud, size: 1 } : [Tone.Calm]; }
  empty(item: Item) { return item.kind === "text" ? { label: undefined, rows: [] } : { label: "", rows: [null] }; }
  other(item: Item) { return item.kind === "text" ? settings.width : this.scale; }
  mixed(item: Item) { return item.kind === "text" ? this.wrap(item) : Math.max(item.count, settings.width); }
  local(item: Item) { return item.kind === "text" ? [1, 2].map((n) => n * 2) : JSON.stringify(item.sku); }
  wrap(item: Item) { return item; }
}`, Files: files, Options: only("decision"), Errors: facts("decision", 5)},
			{Code: `import type { Box, Size } from "./items.js";
export function f(size: Size, ready: boolean) {
  if (size === "S" || (size === "M" && ready)) return 1;
  if (size !== "L" || ready) return 2;
  return 3;
}
export function g(box?: Box) {
  if (!box || box.size === "L") return 0;
  if (box.size === "S") return 1;
  return 2;
}
declare function log(): void;
export function h(size: Size) {
  if (size === "L") return 0;
  log();
  if (size === "S") return 1;
  else return 2;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 4)},
			{Code: `import type { Size } from "./items.js";
declare function next(): Size;
export function f(size: Size, other: { size: Size } | undefined) {
  if (size === "S") return 0;
  size = next();
  if (size === "S") return 1;
  return other?.size === "M" || other!.size === "L" ? 2 : 3;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 3)},
			{Code: `import type { Size } from "./items.js";
const declared: readonly Size[] = ["S", "M"];
const literal = ["S", "M"] as const;
export function f(size: Size) {
  if (declared.includes(size)) return 1;
  if (literal.includes(size)) return 2;
  return 0;
}`, Files: files, Options: only("test"), Errors: facts("test", 1)},
			{Code: `import type { Item } from "./items.js";
export type Panel = { size: "S" | "L"; title: string };
export function f({ kind }: Item, { title, ...rest }: Panel) {
  return kind === "text" && title ? rest.size : undefined;
}
export function g(panel: Readonly<Partial<Panel>>) {
  return panel.size === "S" ? 1 : panel.size === "L" ? 2 : 0;
}`, Files: files, Options: only("test", "decision"), Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "decision"}, {MessageId: "decision"}, {MessageId: "test"}, {MessageId: "test"}, {MessageId: "test"}, {MessageId: "test"},
			}},
			{Code: `import type { Size } from "./items.js";
const Labels = { of(text: string) { return text.toUpperCase(); } };
const loading = Promise.resolve("loading");
export function f(size: Size) {
  return size === "S" ? Labels.of("small") : size === "M" ? loading : JSON.stringify(size);
}`, Files: files, Options: only("decision"), Errors: facts("decision", 1)},
			{Code: `export class Door {
  #state: "open" | "shut" = "shut";
  #lock?: string;
  label() { return this.#state === "open" ? "open" : this.#lock ? "locked" : "shut"; }
}`, Options: only("test"), Errors: facts("test", 2)},
			{Code: `import type { Size } from "./items.js";
export class Panel {
  constructor(readonly size: Size) {}
  width() { return this.size === "S" ? 10 : this.size === "M" ? 20 : 30; }
}
export function outside(panel: Panel) {
  return panel.size === "S" ? { narrow: panel.width() } : { wide: panel.width() };
}`, Files: files, Options: only("decision"), Errors: facts("decision", 2)},
			{Code: `import type { Item } from "./items.js";
declare function show(text: string): string;
export function f(item: Item) { return item.kind === "text" ? show(item.text) : ""; }
export function g(text: string | undefined) { return text ? show(text) : ""; }
export function h(item: Item | undefined) { return item !== undefined && item.kind === "stock" && show(item.sku); }`, Files: files, Options: only("guard"), Errors: facts("guard", 4)},
			{Code: `import type { Item } from "./items.js";
export function f(text: string | undefined) {
  if (!text) return [];
  return [text];
}
export function g(item: Item) {
  if (item.kind === "stock") { console.log(item.sku); }
}
export async function h(text?: string) {
  if (!text) return null;
  return text;
}
export const k = (text: string | null) => text !== null && text.length > 2;`, Files: files, Options: only("entry"), Errors: facts("entry", 4)},
			{Code: `function strict(text: string) { return text.length; }
function loose(text: string | undefined) {
  if (!text) return 0;
  return text.length;
}
function fallback(text = "") { return text.length; }
export function f(text: string | undefined) {
  const a = text ? strict(text) : 0;
  const b = text ? loose(text) : 0;
  const c = text ? fallback(text) : 0;
  return a + b + c;
}`, Options: only("boundary"), Errors: facts("boundary", 3)},
			{Code: `declare function validate(value: string): void;
class Rejected extends Error { constructor(readonly input: string) { super(input); } }
export function parse(input: string, limit: number) {
  if (input.length > limit) throw new Rejected(input);
  validate(input);
  const trimmed = input.trim();
  return { value: trimmed, limit };
}
export class Account {
  private balance = 0;
  deposit(amount: number) {
    if (amount <= 0) throw new Error("positive");
    this.balance += amount;
  }
}`, Options: only("flow"), Errors: facts("flow", 3)},
			{Code: `type Pet = { kind: "cat"; lives: number } | { kind: "dog"; good: boolean };
export function isCat(pet: Pet): pet is Extract<Pet, { kind: "cat" }> { return pet.kind === "cat"; }
export function isOldCat(pet: Pet): pet is Extract<Pet, { kind: "cat" }> { return pet.kind === "cat" && pet.lives !== 9; }
export function isTabby(pet: Pet): pet is Extract<Pet, { kind: "cat" }> { return pet.kind === "dog"; }`, Options: only("contract"), Errors: facts("contract", 1)},
			{Code: `import type { Item, StockItem } from "./items.js";
export const view = {
  matches: (item: Item): item is StockItem => item.kind === "stock",
  render(item: Item) {
    if (item.kind !== "stock") return undefined;
    return item.sku;
  },
};`, Files: files, Options: only("function", "gate"), Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "function"}, {MessageId: "function"}, {MessageId: "gate"},
			}},
			{Code: `type Node = { status: "running" | "failed" | "done" };
declare function paint(color: "accent" | "error" | "dim" | "success", text: string): string;
export function row(node: Node) {
  const a = node.status === "failed" ? "error" : node.status === "running" ? "accent" : "dim";
  const b = node.status === "running" ? "accent" : node.status === "failed" ? "error" : "success";
  return paint(a, "x") + paint(b, "y");
}`, Options: only("conflict"), Errors: facts("conflict", 1)},
		},
	)
}
