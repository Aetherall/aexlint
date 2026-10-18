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
			{Code: `declare function isRecord(value: unknown): value is Record<string, unknown>;
export function decode(value: unknown, input: unknown, record: { kind: unknown; name: unknown }) {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("object");
  if (!input || !isRecord(input)) throw new Error("record");
  const valid = typeof record.kind === "number" && typeof record.name === "string";
  if (!valid) throw new Error("fields");
  return [value, input, record];
}`, Options: only("rejection")},
			{Code: `import type { Item } from "./items.js";
export const isPlain = (item: Item) => !(item.kind === "text" && item.text.length > 0);`, Files: files, Options: only("function")},
			{Code: `export function f(input: string) {
  const a: boolean = !a || input === "";
  const b: boolean = c || input === "x";
  const c: boolean = b;
  if (a || b) return input;
  return "";
}`, Options: only("rejection")},
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
			{Code: `import type { Size } from "./items.js";
export const open = (size: Size) => (size === "S" || size === "M") satisfies boolean;
export const shut = (size: Size) => (size === "L") as boolean;`, Files: files, Options: only("site"), Errors: facts("site", 2)},
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
			{Code: `import { type Item, log } from "./items.js";
let last = "";
export function label(item: Item) {
  let text = "";
  switch (item.kind) {
    case "text": text = "Text"; break;
    case "stock": text = item.sku; log(text); break;
  }
  return text;
}
export function remember(item: Item) {
  if (item.kind === "text") last = "Text";
  else last = item.sku;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 2)},
			{Code: `import type { Size } from "./items.js";
declare const current: Size;
export function f(size: Size) {
  switch (size) {
    case "S":
      log("small");
    case "M":
      log("medium");
      break;
    case current:
      log("current");
      break;
    default:
      log("other");
  }
}
declare function log(text: string): void;`, Files: files, Options: only("decision"), Errors: facts("decision", 1)},
			{Code: `type Trim = { kind?: "a" | "b" | "c" };
declare function log(text: string): void;
export function f(trim: Trim) {
  if (trim.kind === "a") log("a");
  else if (trim.kind === "b") log("b");
  else if (trim.kind === "c") log("c");
  else log("none");
}`, Options: only("decision"), Errors: facts("decision", 1)},
			{Code: `import type { Item } from "./items.js";
export function total(item: Item, prices: number[]) {
  let sum = 0;
  if (item.kind === "text") {
    sum += 2;
  } else {
    let each = 0;
    each = prices.length;
    sum = each;
  }
  return sum;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 1)},
			{Code: `import type { Size } from "./items.js";
declare function log(text: string): void;
export function f(sizes: Size[], size: Size, ready: boolean) {
  for (const each of sizes) if (each === "S") log("small"); else if (each === "M") log("medium");
  if (ready) if (size === "S") log("small"); else log("large");
  outer: if (size === "L") log("large"); else if (size === "M") log("medium");
}`, Files: files, Options: only("decision"), Errors: facts("decision", 3)},
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
declare function fail(reason: string): never;
export function f(size: Size, flag: boolean) {
  if (size === "S") { fail("small"); }
  if (size === "M") { if (flag) return 1; else throw new Error("m"); }
  return size === "L" ? 2 : 3;
}
export function g(size: Size, mode: "a" | "b") {
  if (size === "S") {
    switch (mode) { case "a": return 1; case "b": throw new Error("b"); }
  }
  if (size === "M") {
    switch (mode) { case "a": if (mode) break; return 1; default: return 2; }
  }
  return size === "L" ? 3 : 4;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 6)},
			{Code: `import type { Item } from "./items.js";
type Ponds = { pond: { id?: string; names: Record<string, string> } };
export function pick(item: Item) {
  const { kind } = item;
  if (kind === "text") return 1;
  return item.kind === "stock" ? 2 : 3;
}
export const current = ({ pond }: Ponds) => (pond.id ? pond.names[pond.id] : undefined);
export function first([head]: [string | undefined, number]) {
  if (!head) return 0;
  return head.length;
}
export const point = ([x, y, z]: [number, number] | [number, number, number]) => (z === undefined ? [x, y] : [x, y, z]);
export function moved({ pond }: Ponds) {
  if (!pond.id) return undefined;
  pond = { names: {} };
  return pond.names;
}`, Files: files, Options: only("decision", "entry"), Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "decision"}, {MessageId: "decision"}, {MessageId: "entry"}, {MessageId: "entry"}, {MessageId: "entry"},
			}},
			{Code: `import type { Item } from "./items.js";
export function render({ kind, ...rest }: Item) {
  switch (kind) {
    case "text": return { label: "text", parts: [rest] };
    case "stock": return { label: "stock", parts: [rest] };
  }
}
export function describe(item: Item) {
  const tag = item.kind;
  return tag === "text" ? { text: item.text } : { sku: item.sku, count: item.count };
}
export function plain(item: Item) {
  const { kind } = item;
  return kind === "text" ? { text: item.text } : { sku: item.sku };
}`, Files: files, Options: only("decision"), Errors: facts("decision", 3)},
			{Code: `import type { Item } from "./items.js";
export class Shelf {
  constructor(readonly item: Item) {}
  label() {
    const { kind } = this.item;
    return kind === "text" ? { text: this.item } : { stock: this.item };
  }
}`, Files: map[string]string{"items.ts": items}, Options: only("decision"), Errors: facts("decision", 1)},
			{Code: `export function save({ id, name }: { id: string; name: string }) {
  const short = id;
  if (short.length > 10) throw new Error("long");
  return name;
}`, Options: only("rejection"), Errors: facts("rejection", 1)},
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
			{Code: `declare function cut(text: string, limit: number): string;
type Row = { id: string };
export const label = ({ limit, text }: { limit?: number; text: string }) => (limit ? cut(text, limit) : text);
export const rows = (list: Row[], id: string | undefined) => (id !== undefined ? list.filter((row) => row.id === id) : []);
export const known = (id: "a" | "b" | undefined) => (id !== undefined ? ["a", "b"].includes(id) : false);`, Options: only("guard"), Errors: facts("guard", 2)},
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
			{Code: `function loose(text: string): number;
function loose(text: string | undefined): number;
function loose(text: string | undefined) {
  if (!text) return 0;
  return text.length;
}
export function f(text: string | undefined) {
  const length = text ? loose(text) : 0;
  return length;
}`, Options: only("boundary"), Errors: facts("boundary", 1)},
			{Code: `function check(value: string): string;
function check(value: number): number;
function check(value: string | number) {
  if (value === "") throw new Error("empty");
  return value;
}
export function parse(input: string) { return check(input); }`, Options: only("rejection"), Errors: facts("rejection", 2)},
			{Code: `export function upgrade(input: { version: number; name: string }, flag: boolean) {
  if (flag) return input;
  if (input.version === 2) return input;
  if (input.name !== "") {
    log(input.name);
  } else {
    return input;
  }
  throw new Error("unsupported");
}
declare function log(text: string): void;`, Options: only("rejection"), Errors: facts("rejection", 1)},
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
			{Code: `export class Label {
  constructor(readonly value: string) {
    if (value.length > 47) throw new Error("label");
  }
}`, Options: only("flow"), Errors: facts("flow", 1)},
			{Code: `declare function validate(value: string): void;
export function parse(input: string, limit: number) {
  if (input.length > limit) throw new Error("long");
  if (Date.now() > 0) throw new Error("late");
  try { if (!input) throw new Error("empty"); } catch {}
  try { if (input === "x") throw new Error("reserved"); } finally { validate(input); }
  return input;
}`, Options: only("rejection"), Errors: facts("rejection", 2)},
			{Code: `export function decode(value: unknown, name: string, count: number) {
  if (!value) throw new Error("present");
  const long = name.length > 3;
  if (!long) throw new Error("long");
  if (!count && typeof count !== "object") throw new Error("count");
  if (!/^[a-z]+$/.test(name)) throw new Error("pattern");
  return [value, name, count];
}`, Options: only("rejection"), Errors: facts("rejection", 4)},
			{Code: `export function read(input: { kind: "a" | "b"; extra?: string }) {
  if (input.kind === "a") { if (input.extra !== undefined) throw new Error("then"); }
  if (input.kind === "b") { if (input.extra === undefined) {} else throw new Error("else"); }
  return input;
}`, Options: only("rejection"), Errors: facts("rejection", 2)},
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
			{Code: `import type { Item } from "./items.js";
const filled = (item: Item) => item.kind === "text" && item.text.length > 0;
export const unfilled = (item: Item) => !filled(item);
export const notText = (item: Item) => !(item.kind === "text");`, Files: files, Options: only("function"), Errors: facts("function", 2)},
			{Code: `import type { Size } from "./items.js";
declare function log(text: string): void;
export function run(size: Size) {
  if (size === "S") return 1;
  if (size === "M") return 2;
  return 3;
}
export function cases(size: Size) {
  switch (size) {
    case "S":
      return "small";
  }
  return "other";
}
export function broken(size: Size) {
  switch (size) {
    case "S":
      break;
    case "M":
      return "medium";
  }
  return "other";
}
export function stays(size: Size) {
  if (size === "S") log("small");
  if (size === "M") return 2;
  return 3;
}
export function works(size: Size) {
  if (size === "S") return 1;
  log("other");
  return 3;
}`, Files: files, Options: only("decision"), Errors: facts("decision", 5)},
			{Code: `declare const raw: string;
type Level = "low" | "high" | "max";
declare const current: Level;
export function indicator(value: string | undefined) {
  switch (value) {
    case "low":
      return 1;
    case "high":
      return 2;
  }
  return 0;
}
export function open(value: string) { return value === "low"; }
export const stored = (value: string) => ["low", "high"].includes(value);
export function renamed(value: string) { return value === "max"; }
export class Panel {
  #mode(value: string) { return value === "high"; }
  run() { return this.#mode(current); }
}
class Base { mode(value: string) { return value.length; } }
export class Child extends Base { override mode(value: string) { return value === "max" ? 1 : 0; } }
export function typed(value: Level) { return value === "max"; }
open(raw);`, Files: map[string]string{"callers.ts": `import { indicator, open, stored, renamed as again, typed } from "./file.js";
declare const level: "low" | "high" | "max" | undefined;
indicator(level);
indicator("low");
open("low");
export const handlers = [stored];
again("max");
typed("max");`}, Options: only("callers"), Errors: facts("callers", 7)},
			{Code: `type Level = "low" | "high" | "max";
interface Meter { read(value: string): number }
export class Gauge {
  read(value: string) { return value === "high" ? 1 : 0; }
}
export class Dial {
  read(value: string) { return value === "low" ? 1 : 0; }
}
class Base { tick(value: string) { return value.length; } }
export class Clock extends Base { override tick(value: string) { return value === "max" ? 1 : 0; } }
export class Timer implements Meter { read(value: string) { return value === "max" ? 1 : 0; } }
declare const meter: Meter;
declare const level: Level;
declare const raw: string;
meter.read(raw);
new Gauge().read(level);
new Dial().read("low");
new Dial().read("high");`, Options: only("callers"), Errors: facts("callers", 4)},
			{Code: `type Level = "low" | "high" | "max";
declare const level: Level;
declare const raw: string;
function pick(value: string) { return value === "low" ? 1 : 2; }
export function name(flag: boolean): string { return flag ? level : "low"; }
export function plain(flag: boolean): string { return flag ? raw : "low"; }
export const arrow = (): string | undefined => level;
export async function later(): Promise<string> { return level; }
if (name(true) === "max") pick("low");
const named = arrow();
if (named === "high") pick("high");
pick(name(false));
plain(true);`, Options: only("result"), Errors: facts("result", 3)},
			{Code: `export class Label {
  constructor(readonly value: string) {}
  static parse(value: string) { return new Label(value); }
}
export async function use(value: string) {
  new Label(value);
  (await Promise.resolve(Label.parse(value)));
  const kept = Label.parse(value);
  return [kept, Label.parse(value)];
}`, Options: only("flow"), Errors: facts("flow", 2)},
			{Code: `export interface Order { state: string }
export type Line = { count: number };
export class Cart { total = 0 }
export class Special extends Cart { extra?: number }
export const Anonymous = class { hidden = 1 };
declare global { interface Ambient { shade: string } }
export namespace Scope { export interface Inner { depth: number } }
export function f(value: { loose: string }) { return value; }`, Options: only("owner"), Errors: facts("owner", 8)},
		},
	)
}
