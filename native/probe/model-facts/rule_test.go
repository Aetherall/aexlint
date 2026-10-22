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

const summarized = `declare function use(...values: unknown[]): void;
class Badge {
  active = false;
}
class Club {
  readonly badges: Badge[] = [];
  readonly byId = new Map<string, Badge>();
  add(badge: Badge) {
    this.badges.push(badge);
  }
  index(id: string, badge: Badge) {
    this.byId.set(id, badge);
    this.add(badge);
  }
  name(): string {
    return "club";
  }
  at(i: number) {
    return this.badges[i];
  }
  count() {
    return this.badges.length + this.byId.size;
  }
  all() {
    return this.badges.map((badge) => badge.active);
  }
  log(badge: Badge) {
    use(badge);
  }
}
class Shared extends Club {
  constructor(first: Badge) {
    super();
    super.add(first);
  }
}
function activate(badge: Badge, on: boolean) {
  badge.active = on;
}
function fill(target: Badge[], count: number) {
  for (let i = 0; i < count; i++) target.push(new Badge());
}
function local() {
  const mine: Badge[] = [];
  mine.push(new Badge());
  return mine;
}
function rebinds(list: Badge[]) {
  list = [];
  list.push(new Badge());
}
function loop(n: number, club: Club, badge: Badge): void {
  if (n > 0) loop(n - 1, club, badge);
  club.add(badge);
}
export function calls(club: Club, badge: Badge, list: Badge[]) {
  club.add(badge);
  club.index("a", badge);
  club.name();
  club.log(badge);
  activate(badge, true);
  fill(list, 2);
  local();
  rebinds(list);
  loop(2, club, badge);
  new Shared(badge);
}`

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
const sizes = new Set<Size>(["S", "M"]);
declare function load(): readonly Size[];
const loaded = load();
export function f(size: Size) {
  if (declared.includes(size)) return 1;
  if (literal.includes(size)) return 2;
  if (sizes.has(size)) return 3;
  if (loaded.includes(size)) return 4;
  return 0;
}`, Files: files, Options: only("test"), Errors: facts("test", 3)},
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
			{Code: `import type { Item } from "./items.js";
export interface Summary { summary?: string; warning?: string; usage?: number }
declare const maybe: string | undefined;
declare const loose: any;
export function summarize(mode: "a" | "b" | "c"): Summary {
  if (mode === "a") return { summary: "done" };
  if (mode === "b") return { warning: undefined, summary: maybe };
  return { ...loose, usage: 3 };
}
export const link = (ok: boolean): { label?: string; url?: string } | undefined => (ok ? { label: "x", url: "y" } : {});
export async function later(): Promise<Summary> { return { summary: "a" }; }
export function absent(ok: boolean): Summary | null { if (!ok) return null; return { warning: "w" }; }
export function one(): Summary { return { summary: "a" }; }
export function other(item: Item): Summary { if (item.kind === "text") return { summary: item.text }; return mk(); }
declare function mk(): Summary;
export function either(ok: boolean): { a: string } | { b?: string; c?: string } { return ok ? { a: "x" } : { b: "y" }; }
export function inferred(ok: boolean) { return ok ? { a: "x" } : { b: "y" }; }`, Files: files, Options: only("optional"), Errors: facts("optional", 3)},
			{Code: `export interface Summary { summary?: string; warning?: string; usage?: number }
declare function remote(): Promise<{ summary: string; usage: number }>;
declare function partial(): { summary?: string; warning: string | undefined };
declare const list: string[];
export async function summarize(mode: "a" | "b"): Promise<Summary> {
  if (mode === "a") return remote();
  if (mode === "b") return await remote();
  return partial();
}
export function listed(ok: boolean): Summary { return ok ? { summary: "a" } : (list as never); }`, Options: only("optional"), Errors: facts("optional", 1)},
			{Code: `export type Keys = { fieldKey: string | undefined; subFieldKey: string | undefined; always: string };
export function keys(input: string): Keys {
  if (input === "") return { fieldKey: undefined, subFieldKey: undefined, always: input };
  return { fieldKey: input, subFieldKey: input, always: input };
}
export function inferred(input: string) {
  if (input === "") return { fieldKey: undefined, subFieldKey: undefined };
  if (input === " ") return { fieldKey: undefined, subFieldKey: undefined };
  return { fieldKey: input, subFieldKey: input };
}`, Options: only("optional"), Errors: facts("optional", 1)},
			{Code: `import type { Size } from "./items.js";
const small = new Set<Size>(["S", "M"]);
const wide: Record<Size, boolean> = { S: false, M: true, L: true };
export const isSmall = (size: Size) => small.has(size);
export function ship(size: Size) {
  if (size !== "L") return;
  log(size);
}
declare function log(size: Size): void;`, Files: files, Options: only("site"), Errors: facts("site", 3)},
			{Code: `declare function fail(reason: string): never;
declare function use(...values: unknown[]): void;
export function place(rows: Map<number, string>, fits: boolean, own: boolean) {
  let index = 0;
  let row = rows.get(index) ?? "";
  if (!fits) {
    if (!own) return;
    index = rows.size;
    row = "";
  }
  use(row, index);
  let label = "a";
  if (own) label = "b";
  else label = "c";
  use({ label });
  let skipped = 0;
  if (fits) {
    skipped = 1;
    return;
  }
  if (own) {
    skipped = 2;
    fail("own");
  }
  let total = 0;
  if (own) {
    for (const value of rows.values()) {
      if (!value) {
        total++;
        continue;
      }
      skipped++;
    }
  }
  use(total + skipped);
  if (fits) {
    const inner = 1;
    rows.forEach(() => {
      total = inner;
    });
  }
  use(total);
  if (own) total = 3;
  total = 4;
  use(total);
  let first = 0;
  let last = 0;
  if (own) {
    first = 1;
    last = 2;
  }
  return [first, last];
}`, Options: only("left"), Errors: facts("left", 2)},
			{Code: `declare const source: { a: number; b: number[] };
export class Counter {
  count = 0;
  items: number[] = [];
  bump(step: number, pairs: [number, number][]) {
    let a = 0;
    let b = 0;
    a = step;
    a += 2;
    b++;
    [a, b] = [b, a];
    ({ a, b: this.count } = source);
    for ([a, b] of pairs) {
      continue;
    }
    this.items.push(a);
    const n = this.items.pop();
    this.reset();
    return n;
  }
  reset() {}
}`, Options: only("write"), Errors: facts("write", 12)},
			{Code: `declare function fail(): never;
export function f(xs: number[], mode: "a" | "b") {
  outer: for (const x of xs) {
    for (;;) {
      if (x > 1) break outer;
      if (x > 2) continue outer;
      if (x > 3) break;
    }
  }
  switch (mode) {
    case "a":
      return 1;
    case "b":
      fail();
  }
}
export function g(flag: boolean) {
  try {
    if (flag) throw new Error("flag");
    return 1;
  } catch {
    return 2;
  } finally {
    flag = false;
  }
}
export function h(flag: boolean) {
  while (true) {
    if (flag) return 3;
  }
}`, Options: only("ending"), Errors: facts("ending", 11)},
			{Code: `declare function use(...values: unknown[]): void;
export class Range {
  start = 0;
  end = 0;
  set(from: number, to: number, swap: boolean) {
    if (swap) {
      this.start = to;
      this.end = from;
    }
    use(this.start, this.end);
  }
}
export function search(items: string[], wanted: string, out: string[]) {
  let found = false;
  let index = -1;
  for (let i = 0; i < items.length; i++) {
    if (items[i] === wanted) {
      found = true;
      index = i;
      break;
    }
  }
  use(found, index);
  switch (wanted) {
    case "a":
      out.push("a");
      wanted = "b";
      break;
    default:
      wanted = "c";
  }
  use(out, wanted);
  let first = 0;
  let last = 0;
  if (found) {
    first = 1;
    last = 2;
  }
  if (index > 0) {
    first = 3;
    last = 4;
  } else {
    first = 5;
    last = 6;
  }
  use(first, last);
  let [low, high] = [0, 0];
  wanted.length > 2 ? ((low = 1), (high = 2)) : (low = 3);
  use(low, high);
  if (found) {
    out.pop();
  }
  use(out);
  if (found) {
    const n = out.pop();
    use(n);
  }
  use(out);
}`, Options: only("left"), Errors: facts("left", 5)},
			{Code: `declare function use(...values: unknown[]): void;
export function steer(rows: Map<number, string[]>, anchors: number[], items: string[], tree: Map<string, string[]>) {
  let current: string[] = [];
  let size = 0;
  for (const item of items) {
    let index = anchors[0] ?? 0;
    let row = rows.get(index) ?? [];
    if (row.length > 3) {
      index = rows.size;
      row = [];
    }
    row.push(item);
    rows.set(index, row);
    if (size > 3) {
      current = [];
      size = 0;
    }
    current.push(item);
    size += 1;
    let entry = tree.get(item);
    if (!entry) {
      entry = [];
      tree.set(item, entry);
    }
    entry.push(item);
  }
  use(current);
}`, Options: only("merge"), Errors: facts("merge", 3)},
			{Code: `declare function use(...values: unknown[]): void;
class Recorder {
  record(value: number) {}
  total(): number {
    return 0;
  }
}
export function effects(rows: Map<string, number>, list: number[], seen: Set<number>, box: { count: number; items: number[] }, key: string) {
  rows.set(key, 1);
  rows.get(key);
  list.push(2);
  list.map((x) => x);
  list.pop();
  seen.add(3);
  box.count = 4;
  box.count++;
  list[key.length] = 5;
  box.items.sort();
  const recorder = new Recorder();
  recorder.record(6);
  const total = recorder.total();
  use(box, total);
}`, Options: only("effect"), Errors: facts("effect", 8)},
			{Code: `declare function use(...values: unknown[]): void;
enum Slot { Head, Tail }
declare const nothing: undefined;
class Panel {
  items: string[] = [];
}
export function checked(rows: Map<number, string[]>, list: string[], key: number, entry: string[] | undefined, shape: Panel | Error, other: Panel, store: Record<number, string[]>) {
  let index = key;
  if (key > 1) {
    index = Slot.Tail;
  }
  rows.set(index, list);
  let fresh = [] as string[];
  let target = list;
  if (key > 2) {
    target = fresh;
  }
  target.push("a");
  if (entry === nothing) {
    entry = [];
  }
  entry.push("b");
  let panel = other;
  if (shape instanceof Error) {
    panel = new Panel();
  } else {
    panel = shape;
  }
  panel.items.push("c");
  let bucket = store[key];
  if (!bucket) {
    bucket = [];
    store[key] = bucket;
  }
  bucket.push("d");
  use(index, target, entry, panel);
}
export function keyed<K extends string>(store: Partial<Record<K, string[]>>, key: K) {
  let slot = store[key];
  if (!slot) {
    slot = [];
    store[key] = slot;
  }
  slot.push("e");
}`, Options: only("merge"), Errors: facts("merge", 6)},
			{Code: `declare function use(...values: unknown[]): void;
class Row {
  constructor(readonly text: string) {}
}
function blank(): Row {
  return new Row("");
}
const make = (text: string) => new Row(text);
function either<T>(chosen: boolean, a: T, b: T): T {
  return chosen ? a : b;
}
function slot(rows: Map<number, Row>, index: number) {
  return rows.get(index);
}
function size(rows: Map<number, Row>) {
  return rows.size;
}
function again(n: number): Row {
  return n > 0 ? again(n - 1) : blank();
}
async function load(): Promise<Row> {
  return new Row("loaded");
}
class Sheet {
  #rows: Row[] = [];
  get first(): Row | undefined {
    return this.#rows[0];
  }
}
export async function followed(rows: Map<number, Row>, current: Row, key: number, sheet: Sheet, list: Row[]) {
  let a = current;
  if (key > 1) a = blank();
  use(a);
  let b = current;
  if (key > 2) b = make("b");
  use(b);
  let c = current;
  if (key > 3) c = either(true, current, list[0]!);
  use(c);
  let d = current;
  if (key > 4) d = either(false, current, new Row("d"));
  use(d);
  let e = current;
  if (key > 5) e = slot(rows, key) ?? current;
  use(e);
  let f = key;
  if (key > 6) f = size(rows);
  use(f);
  let g = current;
  if (key > 7) g = again(key);
  use(g);
  let h = current;
  if (key > 8) h = await load();
  use(h);
  let i = current;
  if (key > 9) i = sheet.first ?? current;
  use(i);
  let j = list;
  if (key > 10) j = list.filter((row) => row.text);
  use(j);
}`, Options: only("merge"), Errors: facts("merge", 10)},
			{Code: summarized, Options: only("summary"), Errors: facts("summary", 13)},
			{Code: `declare function origin(value: unknown): void;
declare function use(value: unknown): void;
function pass<T>(value: T): T {
  return value;
}
export function origins(lines: string[], anchors: number[], items: string[]) {
  const rows = new Map<number, string[]>();
  const out = [...lines];
  for (const item of items) {
    let index = anchors[0]!;
    if (item.length > 3) {
      index = out.length;
      out.push("");
    }
    rows.set(index, [item]);
  }
  for (const [key, row] of rows) {
    origin(key);
    origin(row);
  }
  for (const key of rows.keys()) origin(key);
  origin(rows.get(1));
  origin(pass(out.length));
  const shared = new Map<number, string>();
  use(shared);
  for (const [key] of shared) origin(key);
  const list: number[] = [];
  list.push(lines.length);
  for (const value of list) origin(value);
}`, Options: only("origin"), Errors: facts("origin", 7)},
			{Code: `declare function use(...values: unknown[]): void;
declare function compute(): string;
class Lines {
  readonly lines: string[] = [];
  place(text: string, index: number) {
    this.lines[index] = text;
  }
}
export function direct(lines: string[]) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  out[index] = compute();
}
export function readBetween(lines: string[]) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  use(out);
  out[index] = compute();
}
export function readThrough(lines: string[]) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  use(out[index]);
  out[index] = compute();
}
export function readOther(lines: string[], other: number) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  use(out[other]);
  out[index] = compute();
}
export function conditional(lines: string[], fill: boolean) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  if (fill) out[index] = compute();
}
export function computed(lines: string[]) {
  const out = [...lines];
  const index = out.length;
  out.push(compute());
  out[index] = compute();
}
export function shelve(items: string[], anchors: number[]) {
  const output = new Lines();
  const rows = new Map<number, string[]>();
  for (const item of items) {
    let index = anchors[0]!;
    let row = rows.get(index) ?? [];
    if (row.length > 3) {
      index = output.lines.length;
      output.lines.push("");
      row = [];
    }
    use(output.lines[anchors[0]!]);
    row.push(item);
    rows.set(index, row);
  }
  for (const [index, row] of rows) {
    output.place(row.join(" "), index);
  }
  return output;
}
export class Panel {
  title = "";
  width = 0;
  rename(next: string) {
    this.title = "";
    this.width = next.length;
    this.title = next;
  }
  reread(next: string) {
    this.title = "";
    use(this);
    this.title = next;
  }
}
export function cells(grid: Map<string, number>, key: string, other: string) {
  grid.set(key, 0);
  grid.set(other, 1);
  grid.set(key, 2);
}
export function exposed(panel: Panel, next: string) {
  panel.title = "";
  use(next);
  panel.title = next;
}
export async function awaited(panel: Panel, next: Promise<string>) {
  panel.title = "";
  panel.title = await next;
}
export function iterated(a: Panel, b: Panel, c: Panel, d: Panel, items: Iterable<string>, list: string[], next: string) {
  a.title = "";
  for (const item of items) void item;
  a.title = next;
  b.title = "";
  const [first] = items;
  b.title = first ?? next;
  c.title = "";
  const all = [...items];
  c.title = next + all.length;
  d.title = "";
  for (const item of list) void item;
  d.title = next;
}
export class Lens {
  constructor(readonly panel: Panel) {}
  get seen(): number {
    use(this.panel);
    return 1;
  }
  get size(): number {
    return this.panel.title.length;
  }
}
export class Box {
  x = 0;
  get double(): number {
    return this.x * 2;
  }
}
export function doubled(box: Box, next: number): number {
  box.x = 1;
  const d = box.double;
  box.x = next;
  return d;
}
export class Tally {
  x = 0;
  #seen = 0;
  set seen(value: number) {
    this.#seen = value + this.x;
  }
  get seen(): number {
    return this.#seen;
  }
}
export function marked(tally: Tally, next: number): void {
  tally.x = 1;
  tally.seen = 5;
  tally.x = next;
  tally.seen = 1;
  tally.seen = next;
}
export function looked(a: Panel, b: Panel, lens: Lens, next: string) {
  a.title = "";
  void lens.seen;
  a.title = next;
  b.title = "";
  const { seen } = lens;
  b.title = next + seen;
  b.title = "";
  void lens.size;
  b.title = next;
}
export function handed(a: Panel, b: Panel, c: Panel, items: string[], cb: (item: string) => void, run: () => void, next: string) {
  a.title = "";
  items.forEach(cb);
  a.title = next;
  b.title = "";
  run.call(undefined);
  b.title = next;
  c.title = "";
  items.forEach((item) => void item);
  c.title = next;
  b.title = "";
  items.forEach((item) => cb(item));
  b.title = next;
  a.title = "";
  const count = items.map(Number).filter(Boolean).length;
  a.title = next + count;
}
function check(value: string) {
  if (!value) throw new Error("empty");
}
export function guarded(panel: Panel, next: string) {
  panel.title = "";
  check(next);
  panel.title = next;
  const local = new Panel();
  local.title = "";
  check(next);
  local.title = next;
  return local;
}
export function aliased(next: string, keep: (inner: { panel: Panel }) => void) {
  const outer = { inner: { panel: new Panel() }, title: "" };
  outer.inner.panel.title = "";
  const inner = outer.inner;
  keep(inner);
  outer.inner.panel.title = next;
  return outer;
}
export function titled(list: [Panel, Panel], next: string) {
  list[0].title = "";
  const titles = list.map((panel) => panel.title);
  list[0].title = next;
  list[1].title = "";
  const count = list.length;
  list[1].title = next + count;
  return titles;
}
export function promised(a: Panel, b: Panel, run: () => void, next: string) {
  a.title = "";
  void new Promise<void>((resolve) => {
    run();
    resolve();
  });
  a.title = next;
  b.title = "";
  void new Promise<void>((resolve) => resolve());
  b.title = next;
  a.title = "";
  void next.replace(/x/g, (found) => {
    run();
    return found;
  });
  a.title = next;
}
export class Quiet {
  log(): void {}
  tidy(): void {}
}
export class Loud extends Quiet {
  override log(): void {
    use(this);
  }
}
export function logged(a: Panel, b: Panel, quiet: Quiet, next: string) {
  a.title = "";
  quiet.log();
  a.title = next;
  b.title = "";
  quiet.tidy();
  b.title = next;
}
export async function streamed(panel: Panel, lines: AsyncIterable<string>, next: string) {
  panel.title = "";
  for await (const line of lines) use(line.length);
  panel.title = next;
}
function make() {
  const panel = new Panel();
  return { panel, check: () => panel.title.length };
}
export function closed(next: string) {
  const ctx = make();
  ctx.panel.title = "";
  ctx.check();
  ctx.panel.title = next;
}
export function captures(next: string, other: Panel) {
  const panel = new Panel();
  const look = () => panel.title.length;
  const peek = () => other.title.length;
  panel.title = "";
  look();
  panel.title = next;
  const kept = new Panel();
  kept.title = "";
  peek();
  kept.title = next;
  return [panel, kept];
}
export function escapes(lines: string[], keep: (out: string[]) => void) {
  const out = [...lines];
  const index = out.length;
  out.push("");
  keep(out);
  use(index);
  out[index] = compute();
  return out;
}`, Options: only("overwritten"), Errors: facts("overwritten", 18)},
			{Code: `declare function use(...values: unknown[]): void;
export function mixed(items: string[], anchor: number, extra: number[]) {
  const rows = new Map<number, string[]>();
  for (const item of items) {
    let index = anchor;
    const row = rows.get(index) ?? [];
    if (row.length > 3) index = extra.length + rows.size;
    row.push(item);
    rows.set(index, row);
  }
  use(rows.has(anchor));
  for (const [index, row] of rows) use(index, row);
}
export function list(items: string[]) {
  const rows = new Map<number, string>();
  for (const item of items) rows.set(rows.size, item);
  for (const [index, row] of rows) use(index, row);
}
export function addressed(items: string[]) {
  const counts = new Map<string, number>();
  for (const item of items) counts.set(item, (counts.get(item) ?? 0) + 1);
  return [...counts.keys()];
}
export function removed(items: string[], drop: string) {
  const seen = new Map<string, boolean>();
  for (const item of items) seen.set(item, true);
  seen.set(drop.trim(), false);
  seen.delete(drop);
  for (const [key] of seen) use(key);
}
export function tags(items: string[], extra: string) {
  const known = new Set<string>();
  for (const item of items) known.add(item);
  known.add(extra.toUpperCase());
  return items.filter((item) => known.has(item)).length + known.size;
}
export function byName(items: string[], fallback: number) {
  const index: Record<string, number> = {};
  items.forEach((item, i) => use(item, i));
  for (const item of items) index[item] = item.length;
  index[String(fallback)] = fallback;
  return index[items[0]!];
}`, Options: only("unaddressed"), Errors: facts("unaddressed", 3)},
			{Code: `declare function use(...values: unknown[]): void;
declare function notify(): void;
class Lines {
  lines: string[] = [];
  place(text: string, index: number) {
    this.lines[index] = text;
  }
}
export function shelve(items: string[], anchors: number[]) {
  const output = new Lines();
  const rows = new Map<number, string[]>();
  for (const item of items) {
    let index = anchors[0]!;
    let row = rows.get(index) ?? [];
    if (row.length > 3) {
      index = output.lines.length;
      output.lines.push("");
      row = [];
    }
    use(output.lines[anchors[0]!]);
    row.push(item);
    rows.set(index, row);
  }
  for (const [index, row] of rows) {
    output.place(row.join(" "), index);
  }
  return output;
}
export function shout(words: string[]) {
  const loud: string[] = [];
  for (const word of words) loud.push(word.toUpperCase());
  for (const word of loud) use(word);
}
export function wrapped(words: string[]) {
  const boxes: { word: string }[] = [];
  for (const word of words) {
    const box = { word };
    use(box.word);
    boxes.push(box);
  }
  for (const box of boxes) use(box);
}
export function counted(words: string[]) {
  const loud: string[] = [];
  for (const word of words) {
    if (loud.length > 3) break;
    loud.push(word);
  }
  for (const word of loud) use(word);
}
export function logged(words: string[], log: string[]) {
  const loud: string[] = [];
  for (const word of words) {
    if (log.length > 3) continue;
    loud.push(word.toUpperCase());
  }
  for (const word of loud) log.push(word);
}
type Item = { name: string; seen: boolean };
export function marked(items: Item[]) {
  const picked: Item[] = [];
  for (const item of items) {
    picked.push(item);
    notify();
  }
  for (const item of picked) use(item.name);
}
export function kept(items: Item[]) {
  const picked: Item[] = [];
  for (const item of items) if (item.seen) picked.push(item);
  for (const item of picked) use(item.name);
}
export function touched(names: string[]) {
  const picked: Item[] = [];
  for (const name of names) {
    const item = { name, seen: false };
    picked.push(item);
    item.seen = true;
  }
  for (const item of picked) use(item);
}
export function captured(names: string[]) {
  const picked: { name: string }[] = [];
  for (const name of names) {
    const item = { name };
    setTimeout(() => use(item));
    picked.push(item);
  }
  for (const item of picked) use(item);
}
export function distinct(words: string[], extra: string) {
  const seen = new Set<string>();
  for (const word of words) if (!seen.has(word)) seen.add(word);
  seen.add(extra.trim());
  for (const word of seen) use(word);
}
export function moved(items: string[], anchor: number, spare: string[]) {
  const rows = new Map<number, string[]>();
  for (const item of items) {
    let index = anchor;
    let row = rows.get(index) ?? [];
    if (row.length > 3) {
      index = item.length + 100;
      row = spare;
    }
    row.push(item);
    rows.set(index, row);
  }
  for (const [index, row] of rows) use(index, row);
}
export function twice(words: string[]) {
  const loud: string[] = [];
  for (const word of words) loud.push(word);
  for (const word of loud) use(word);
  for (const word of loud) use(word);
}
export function reversed(words: string[]) {
  const loud: string[] = [];
  for (const word of words) loud.unshift(word);
  for (const word of loud) use(word);
}
export function trimmed(words: string[]) {
  const seen = new Set<string>();
  for (const word of words) seen.add(word.trim());
  for (const word of seen) use(word);
}
export class Registry {
  readonly #handles = new Map<string, number>();
  attach(names: string[], open: (name: string) => number) {
    const opened: [string, number][] = [];
    try {
      for (const name of names) opened.push([name, open(name)]);
    } catch (error) {
      this.#handles.clear();
      throw error;
    }
    for (const [name, handle] of opened) this.#handles.set(name, handle);
  }
  reset(names: string[]) {
    const opened: [string, number][] = [];
    for (const name of names) opened.push([name, name.length]);
    this.#handles.clear();
    for (const [name, handle] of opened) this.#handles.set(name, handle);
  }
}
export function tallied(words: string[], out: string[]) {
  const byWord: Record<string, string> = {};
  for (const word of words) byWord[word.toLowerCase()] = word.trim();
  for (const [key, word] of Object.entries(byWord)) out.push(key + word);
}
export function listed(words: string[], out: string[]) {
  const byWord: Record<string, string> = {};
  byWord[words.join(",")] = words.join(" ");
  Object.entries(byWord).forEach(([key, word]) => out.push(key + word));
}
function nonBlank(word: string) {
  if (!word.trim()) throw new Error("blank");
}
export function strict(words: string[], out: string[]) {
  const loud: string[] = [];
  for (const word of words) {
    loud.push(word.toUpperCase());
    nonBlank(word);
  }
  for (const word of loud) out.push(word);
}
export function failing(words: string[], seen: string[]) {
  const loud: string[] = [];
  for (const word of words) {
    loud.push(word.toUpperCase());
    seen.push(word);
  }
  for (const word of loud) nonBlank(word);
}
export function tallies(words: string[]) {
  const loud: string[] = [];
  const counts = new Map<string, number>();
  for (const word of words) {
    loud.push(word.toUpperCase());
    counts.set(word, (counts.get(word) ?? 0) + 1);
  }
  for (const word of loud) nonBlank(word);
  return counts.size;
}
export function sent(words: string[], out: string[]) {
  const loud: string[] = [];
  for (const word of words) loud.push(word.toUpperCase());
  loud.forEach((word) => out.push(word));
}
export async function saved(names: string[], save: (name: string) => Promise<void>) {
  const pending: string[] = [];
  for (const name of names) pending.push(name.trim());
  for (const name of pending) await save(name);
}
export async function loaded(ids: number[], load: (id: number) => Promise<string>, out: string[]) {
  const texts: string[] = [];
  for (const id of ids) texts.push(await load(id));
  for (const text of texts) out.push(text);
}`, Options: only("buffer"), Errors: facts("buffer", 24)},

			{Code: `declare function use(...values: unknown[]): void;
export function ends(lines: string[], anchors: number[]) {
  const indices = lines.flatMap((line, row) => (line ? [row] : []));
  if (!indices.length) return;
  for (const line of lines) use(line, indices.at(0)!, indices.at(-1)!, indices.at(-2)!);
  if (anchors.length > 1) use(anchors.at(1)!, anchors.at(2)!);
  use(anchors.length === 0 ? 0 : anchors.at(0)!);
  use(anchors.length >= 3 && anchors.at(-3)!);
}
export function lookups(counts: Map<string, number>, key: string, other: string) {
  if (!counts.has(key)) return;
  use(counts.get(key)!, counts.get(other)!);
  if (counts.has(other) && counts.get(other)! > 2) use(other);
}
export function changed(items: string[], extra: Map<string, number>, key: string) {
  if (!items.length) return;
  items.pop();
  use(items.at(0)!);
  if (!extra.has(key)) return;
  for (const item of items) {
    use(extra.get(key)!);
    extra.delete(item);
  }
}
export function reordered(items: string[], from: string[]) {
  if (items.length === 0 || from.length < 2) return;
  items.sort();
  use(items.at(0)!, from.at(1)!, from.at(2)!);
}
export function rebound(items: string[]) {
  if (!items.length) return;
  items = [];
  use(items.at(0)!);
}
export function held(items: string[], counts: Map<string, number>, key: string) {
  let count = items.length;
  const known = counts.has(key);
  if (count < 2 || !known) return;
  use(items.at(1)!, counts.get(key)!);
}
declare function assert(condition: unknown, message?: string): asserts condition;
export function asserted(items: string[], counts: Map<string, number>, key: string) {
  assert(items.length > 1, "two items");
  assert(counts.has(key));
  use(items.at(-2)!, counts.get(key)!);
}
function needRows(rows: string[], counts: Map<string, number>, key: string) {
  if (rows.length < 2) throw new Error("two rows");
  if (!counts.has(key)) throw new Error("key");
}
function softRows(rows: string[]) {
  if (!rows.length) return;
}
function draining(rows: string[]) {
  if (!rows.length) throw new Error("rows");
  rows.pop();
}
export function viaCalls(rows: string[], counts: Map<string, number>, key: string) {
  needRows(rows, counts, key);
  use(rows.at(1)!, counts.get(key)!);
  softRows(rows);
  draining(rows);
  use(rows.at(0)!);
}
export function holes(items: (string | undefined)[], counts: Map<string, number | undefined>, key: string) {
  if (!items.length || !counts.has(key)) return;
  use(items.at(0)!, counts.get(key)!);
}`, Options: only("backing"), Errors: facts("backing", 15)},
			{Code: `declare function use(...values: unknown[]): void;
declare function width(text: string): number;
type Note = { text: string; hidden?: boolean; size: number };
export function sum(notes: Note[], limit: number) {
  let total = 0;
  for (const note of notes) {
    if (note.hidden) continue;
    const { text } = note;
    const size = text.length;
    if (size === 0 || size > limit) {
      continue;
    }
    total += size;
  }
  return total;
}
export function each(notes: Note[], out: string[]) {
  notes.forEach((note) => {
    if (note.hidden) return;
    out.push(note.text);
  });
  notes.forEach((note) => {
    if (note.hidden) return note.size;
    out.push(note.text);
    return 0;
  });
}
function sized(note: Note) {
  if (note.size < 0) throw new Error("negative");
  return note.size;
}
export function risky(notes: Note[], out: string[]) {
  for (const note of notes) {
    if (sized(note) > 10) continue;
    out.push(note.text);
  }
}
export function measured(notes: Note[], out: string[]) {
  for (const note of notes) {
    const size = width(note.text);
    if (size > 10) continue;
    out.push(note.text);
  }
}
export function changing(notes: Note[], seen: Set<string>, limit: number) {
  let budget = limit;
  for (const note of notes) {
    if (seen.has(note.text)) continue;
    seen.add(note.text);
  }
  for (const note of notes) {
    if (note.size > budget) continue;
    budget -= note.size;
  }
  for (const note of notes) {
    if (note.hidden) continue;
    note.hidden = true;
  }
}
export function ordered(notes: Note[], total: { count: number }) {
  for (const note of notes) {
    total.count += 1;
    if (note.hidden) continue;
    use(note);
  }
  for (const note of notes) {
    if (note.hidden) break;
    use(note);
  }
  outer: for (const note of notes) {
    for (const other of notes) {
      if (other === note) continue outer;
    }
  }
}`, Options: only("filter"), Errors: facts("filter", 4)},
			{Code: `import type { Note } from "./note.js";
declare function use(...values: unknown[]): void;
export class Panel {
  side?: "left" | "right";
  width: number | undefined;
  get resolved() {
    return this.side ?? "left";
  }
}
type Options = { mode?: "fast" | "slow"; label?: string; count?: number };
export function read(note: Note, panel: Panel, options: Options, loose: { flag?: boolean }) {
  use(note.overflow ?? "hide", note.overflow ?? null, note.text ?? "", note.text);
  use(panel.side ?? "right", panel.width ?? 0, options.mode ?? "fast", options.label ?? "none");
  const { overflow = "own", text = "" } = note;
  use(overflow, text, loose.flag ?? false);
}
export function defaults({ mode = "slow", count = 1 }: Options) {
  return [mode, count];
}
export function normalized(note: Note) {
  note.overflow ??= "own";
  return note;
}`, Files: map[string]string{"note.ts": "export interface Note { text: string; overflow?: \"hide\" | \"own\" }\n"}, Options: only("default"), Errors: facts("default", 10)},
			{Code: `declare function use(...values: unknown[]): void;
class Row {
  constructor(readonly text: string) {}
  render(index: number, width: number) {
    return this.text.padEnd(width) + index;
  }
}
class Listener {
  handle(event: string) {
    use(event);
  }
}
export function rows(lines: string[], width: number) {
  const rows = new Map<number, Row>();
  for (const [index, line] of lines.entries()) {
    const row = rows.get(index) ?? new Row(line);
    rows.set(index, row);
  }
  for (const [index, row] of rows) use(row.render(index, width), index);
  rows.forEach((row, index) => use(row.render(width, index)));
  for (const [index, row] of rows) use(row.render(0, width), index);
}
export function shared(events: string[], listener: Listener, made: Listener[]) {
  const listeners = new Map<string, Listener>();
  for (const event of events) listeners.set(event, listener);
  for (const [event, handler] of listeners) handler.handle(event);
  const own = new Map<string, Listener>();
  for (const event of events) own.set(event, made[0] ?? new Listener());
  for (const [event, handler] of own) handler.handle(event);
}
export function counted(words: string[]) {
  const counts = new Map<string, number>();
  for (const word of words) counts.set(word, (counts.get(word) ?? 0) + 1);
  for (const [word, count] of counts) use(word, count.toFixed(word.length));
}`, Options: only("ownkey"), Errors: facts("ownkey", 2)},
			{Code: `declare function assert(condition: unknown): asserts condition;
export function needRows(rows: string[], counts: Map<string, number>, key: string) {
  if (rows.length < 2) throw new Error("two rows");
  if (!counts.has(key)) throw new Error("key");
}
export function asserting(rows: string[]) {
  assert(rows.length > 0);
}
export function softRows(rows: string[]) {
  if (!rows.length) return;
}
export function draining(rows: string[]) {
  if (!rows.length) throw new Error("rows");
  rows.pop();
}
export function careful(rows: string[], counts: Map<string, number>, key: string) {
  try {
    needRows(rows, counts, key);
  } catch {
    return false;
  }
  return true;
}
export function careless(rows: string[], counts: Map<string, number>, key: string) {
  needRows(rows, counts, key);
  return true;
}
export function each(rows: string[]) {
  rows.forEach((row) => {
    if (!row) throw new Error("blank");
  });
}
export function marked(rows: { done: boolean }[]) {
  rows.forEach((row) => {
    row.done = true;
  });
}
export function gathered(rows: string[], out: string[]) {
  return rows.reduce((all, row) => {
    all.push(row);
    return all;
  }, out);
}
export function looped(rows: { done: boolean }[], byName: Map<string, { done: boolean }>) {
  for (const row of rows) row.done = row.done || true;
  for (const [, row] of byName) row.done = false;
}
export function indexed(rows: { done: boolean }[]) {
  for (const [index, row] of rows.entries()) row.done = index > 0;
}`, Options: only("summary"), Errors: facts("summary", 14)},
			{Code: `type Panel = { title: string; width?: number };
export function assigned(panel: Panel, next: Panel, key: string) {
  Object.assign(panel, next, { width: 1 });
  Object.defineProperty(panel, "title", { value: next.title });
  Reflect.set(panel, key, next);
  Reflect.deleteProperty(panel, key);
  Object.keys(panel);
  Object.freeze(next);
}`, Options: only("effect"), Errors: facts("effect", 4)},
			{Code: `declare function use(...values: unknown[]): void;
class Row {
  items: string[] = [];
  add(item: string) {
    this.items.push(item);
  }
}
export function rows(lines: string[], given: Row, keep: Row[]) {
  const rows = new Map<number, Row>();
  for (const [index, line] of lines.entries()) {
    let row = rows.get(index) ?? new Row();
    if (row.items.length > 3) row = new Row();
    row.add(line);
    rows.set(index, row);
  }
  for (const [, row] of rows) {
    const { items } = given;
    use(row.items.length, items.length);
  }
  const borrowed = given;
  const kept = new Row();
  keep.push(kept);
  const shared = new Row();
  use(shared);
  const closed = new Row();
  const read = () => closed.items.length;
  const fresh = [new Row(), new Row()];
  const sizes = fresh.map((row) => row.items.length);
  try {
    use(sizes);
  } catch (error) {
    use(String(error));
  }
  const parsed = JSON.parse("{}") as { done?: boolean };
  parsed.done = true;
  const copied = Array.from(lines);
  copied.push("");
  return [borrowed, read(), sizes, parsed.done, copied.length];
}`, Options: only("local"), Errors: facts("local", 5)},
			{Code: summarized, Options: only("effect"), Errors: facts("effect", 16)},
			{Code: `declare function use(...values: unknown[]): void;
import * as store from "./store.js";
type Handlers = { notify: (value: string) => void; record(value: string): void };
class Grid extends Map<string, number> {}
export function receivers(either: Map<string, number> | Set<string>, both: Map<string, number> | Grid, handlers: Handlers, key: string, Map: { set(key: string, value: number): void }) {
  both.set(key, 1);
  handlers.notify(key);
  handlers.record(key);
  store.save(key);
  Map.set(key, 2);
  either.clear();
  console.log(key);
  use(either);
}`, Files: map[string]string{"store.ts": "export function save(key: string) {}\n"}, Options: only("effect"), Errors: facts("effect", 4)},
			{Code: `type Vent = { flow?: "low" | "high" | "off"; name?: string };
export function f(vent: Vent, flows: ("low" | "high")[]) {
  if ((vent.flow ?? "low") === "low") return 1;
  if (vent.flow ?? "off" !== "high") return 2;
  if ((vent.name ?? "low") === "low") return 3;
  if ((vent.flow ?? flows[0]) === "high") return 4;
  switch (vent.flow ?? "off") {
    case "off":
      return 5;
    case "low":
    case "high":
      return 6;
  }
}`, Options: only("test", "decision"), Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "decision"}, {MessageId: "decision"}, {MessageId: "test"},
			}},
			{Code: `declare function use(...values: unknown[]): void;
class Target {
  constructor(readonly start: number, readonly end: number) {}
}
class Folder {
  constructor(readonly parent?: Folder) {}
}
export function layout(widths: number[]): Target[] {
  const targets: Target[] = [];
  let column = 0;
  let text = "";
  for (const width of widths) {
    const end = column + width;
    targets.push(new Target(column, end));
    text += " ".repeat(width);
    column = end + 2;
  }
  use(text);
  return targets;
}
export function ancestors(folder: Folder): Folder[] {
  const found: Folder[] = [];
  let current = folder.parent;
  while (current) {
    found.push(current);
    current = current.parent;
  }
  return found;
}
export function mapped(widths: number[]): number[] {
  const doubled: number[] = [];
  for (const width of widths) doubled.push(width * 2);
  return doubled;
}
export function folded(widths: number[]): number {
  let total = 0;
  for (const width of widths) total += width;
  return total;
}
export function counted(widths: number[]): number[] {
  const out: number[] = [];
  let index: number;
  for (index = 0; index < widths.length; index++) out.push(widths[index]! + index);
  return out;
}
export function aside(widths: number[]): number[] {
  const out: number[] = [];
  let seen = 0;
  for (const width of widths) {
    out.push(width);
    seen++;
  }
  use(seen);
  return out;
}`, Options: only("scan"), Errors: facts("scan", 2)},
			{Code: `type Page = { items: string[]; next?: Page };
export function paged(first: Page): string[] {
  const items: string[] = [];
  let page: Page | undefined = first;
  let count = 0;
  while (page) {
    if (count++ > 100) throw new Error("too many pages");
    items.push(...page.items);
    page = page.next;
  }
  return items;
}
export function labelled(names: string[]): string[] {
  const out: string[] = [];
  let index = 0;
  for (const name of names) {
    out.push(index + ": " + name);
    index += 1;
  }
  return out;
}
export function skipping(names: string[]): string[] {
  const out: string[] = [];
  let index = 0;
  for (const name of names) {
    if (!name) continue;
    out.push(index + ": " + name);
    index++;
  }
  return out;
}`, Options: only("scan"), Errors: facts("scan", 3)},
			{Code: `import type { Parcel } from "./parcel.js";
export function slugOf(text: string): string { return text.toLowerCase(); }
export function blank(text: string): boolean { return text === ""; }
export class Shelf { find(label: string): string { return label; } }
export function link(parcel: Parcel): string { return slugOf(parcel.label); }
export function heading(parcel: Parcel): string { return slugOf(parcel.label); }
export function empty(parcel: Parcel): boolean { return blank(parcel.label); }
export function found(parcel: Parcel, shelf: Shelf): string { return shelf.find(parcel.label); }`, Files: map[string]string{"parcel.ts": "export interface Parcel { readonly label: string }\n"}, Options: only("derivation"), Errors: facts("derivation", 2)},
		},
	)
}
