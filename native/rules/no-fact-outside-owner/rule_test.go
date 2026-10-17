package no_fact_outside_owner

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const folder = `export type Action = { at: number };
export class Folder {
  deleted?: Action;
  note?: Action;
  memo?: Action;
  label?: string;
  kind: "draft" | "review" | "final" = "draft";
  constructor(readonly contentType: "image" | "video" | "audio") {}
  isDeleted() {
    return Boolean(this.deleted);
  }
  isDraft() {
    return this.kind === "draft";
  }
  isNotDraft() {
    return this.kind !== "draft";
  }
  isFinal = () => this.kind === "final";
  get hasLabel() {
    return this.label !== undefined;
  }
  isVisual() {
    return this.contentType === "image" || this.contentType === "video";
  }
  getNote() {
    return this.note;
  }
  noMemo() {
    return this.memo === undefined;
  }
}`

const plain = `export type Action = { at: number };
export class Folder {
  deleted?: Action;
  kind: "draft" | "review" | "final" = "draft";
}`

const items = `export type Item = { kind: "text" | "stock" | "note"; text: string };
export type Text = { type: "text"; body: string };
export type Image = { type: "image"; url: string };
export type Leaflet = Text | Image;
export type Seat = { kind: "taken" | "free" };
export type Task = { finishedAt?: { at: number } };
export type Wide = Item & { extra: number };`

func helpers(parameter string) string {
	return `import type { Item, Leaflet, Text, Seat, Task, Wide } from "./items.js";
export function isNonStock(item: ` + parameter + `) {
  return item.kind !== "stock";
}
export class LeafletGuards {
  static isText(m: Leaflet | undefined): m is Text {
    return m !== undefined && m.type === "text";
  }
}
export const SeatUtils = {
  isTaken: (r: Seat) => r.kind === "taken",
};
export const isDone = (t: Task) => t.finishedAt !== undefined;
function isTextItem(item: Item) {
  return item.kind === "text";
}
export function isPlainText(item: Item, strict: boolean) {
  return strict && item.kind === "text";
}
export const check = (w: Wide) => w.kind === "note" || isTextItem(w);`
}

func at(code, snippet string, occurrence int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range occurrence {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "bypassedPredicate", Line: strings.Count(before, "\n") + 1, Column: column, EndColumn: column + len(snippet)}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"folder.ts": folder}
	header := `import { Folder } from "./folder.js";
`
	// The predicate's field is generic; the tested field is instantiated, so their declared types hold different values.
	instantiated := `class Box<T> {
  constructor(readonly value: T | undefined) {}
  isEmpty() { return this.value === undefined; }
}
export const f = (box: Box<string[]>) => (box.value ? "full" : "empty");`
	invalid := func(code, snippet string, occurrence int) rule_tester.InvalidTestCase {
		return rule_tester.InvalidTestCase{Code: code, Files: files, Errors: []rule_tester.InvalidTestCaseError{at(code, snippet, occurrence)}}
	}
	truthy := header + `export function f(folder: Folder | undefined) { if (!folder || folder.deleted) throw new Error("missing"); }`
	filter := header + `export const f = (folders: Folder[]) => folders.filter((f) => !f.deleted);`
	compared := header + `export const f = (f: Folder) => f.deleted !== undefined;`
	boolean := header + `export const f = (f: Folder) => Boolean(f.deleted);`
	values := header + `export const f = (f: Folder) => f.kind !== "draft";`
	unnarrowed := header + `export const f = (f: Folder) => ({ deleted: f.deleted !== undefined, at: f.deleted?.at });`
	arrow := header + `export const f = (f: Folder) => f.kind === "final";`
	getter := header + `export const f = (f: Folder) => (f.label !== undefined ? 1 : 2);`
	chain := header + `export const f = (f: Folder) => f.contentType === "image" || f.contentType === "video";`
	includes := header + `export const f = (f: Folder) => ["image", "video"].includes(f.contentType);`
	alias := `import type { Folder as Entry } from "./folder.js";
export const f = (entry: Entry) => entry.kind === "draft";`
	generic := header + `export const f = <T extends Folder>(f: T) => f.kind === "draft";`
	destructured := header + `export function f(folder: Folder) { const { kind } = folder; return kind === "final"; }`
	optional := header + `export const f = (f?: Folder) => (f?.deleted ? "gone" : "here");`
	missingMemo := header + `export const f = (f?: Folder) => f?.memo !== undefined;`
	nested := header + `export const f = (f: Folder, on: boolean) => on && (f.contentType === "image" || f.contentType === "video");`
	negatedChain := header + `export const f = (f: Folder) => f.contentType !== "image" && f.contentType !== "video";`
	guarded := header + `export const f = (f: Folder | undefined) => !f || f.kind === "draft";`
	shapes := `class Shape {
  constructor(readonly kind: "circle" | "square") {}
  isCircle()%s {
    return this.kind === "circle";
  }
}
class Circle extends Shape { radius = 1; }
class Square extends Shape { side = 1; }
export const f = (s: Circle | Square) => s.kind === "circle";`
	predicate := strings.Replace(shapes, "%s", ": this is Circle", 1)
	helped := map[string]string{"items.ts": items, "helpers.ts": helpers("Item")}
	use := `import type { Item, Leaflet, Seat, Task } from "./items.js";
`
	nonStock := use + `export const f = (item: Item) => (item.kind === "stock" ? 1 : 2);`
	text := use + `export const f = (m: Leaflet) => (m.type === "text" ? m.body : m.url);`
	taken := use + `export const f = (seat: Seat) => (seat.kind === "taken" ? 1 : 2);`
	done := use + `export const f = (tasks: Task[]) => tasks.filter((task) => !task.finishedAt);`
	maybeSeat := use + `export const f = (seat?: Seat) => (seat?.kind === "taken" ? 1 : 2);`
	maybeText := use + `export const f = (m?: Leaflet) => (m?.type === "text" ? 1 : 2);`
	maybeNotText := use + `export const f = (m?: Leaflet) => (m?.type !== "text" ? 1 : 2);`
	image := use + `export const f = (m: Leaflet) => (m.type === "image" ? m.url : m.body);`
	directive := func(statement string) map[string]string {
		return map[string]string{"items.ts": items, "helpers.ts": `import type { Image, Leaflet } from "./items.js";
export function isImage(m: Leaflet): m is Image {
  ` + statement + `
  return m.type === "image";
}`}
	}
	overridden := `class Value {
  constructor(readonly value: string | undefined) {}
  isEmpty() { return this.value === undefined; }
}
class Text extends Value {
  isEmpty() { return this.value === undefined || this.value.trim() === ""; }
}
class Count extends Value {}
export const f = (v: Text | Count) => (v.value === undefined ? 0 : 1);
export const g = (v: Value) => (v.value === undefined ? 0 : 1);`
	stored := header + `export const f = (f?: Folder) => ({ gone: f?.deleted !== undefined });`
	unanswered := func(code, snippet string) rule_tester.InvalidTestCaseError {
		e := at(code, snippet, 1)
		e.MessageId = "unansweredFact"
		return e
	}
	message := `export interface Leaflet {
  postedAt?: { at: number };
  status: "draft" | "sent" | "failed";
  archived?: boolean;
}`
	asked := func(test string) string {
		return `import type { Leaflet } from "./file.js";
export const f = (m: Leaflet) => ({ shown: ` + test + ` });`
	}
	askers := map[string]string{
		"a.ts": asked("!!m.postedAt"),
		"b.ts": `import type { Leaflet as Entry } from "./file.js";
export const f = (m: Entry) => ({ shown: m.postedAt !== undefined ? 1 : 2 });`,
	}
	statuses := map[string]string{"a.ts": asked(`m.status === "draft"`), "b.ts": asked(`(m.status !== "draft" ? 1 : 2)`)}
	order := `export class Order {
  cancelled?: { at: number };
}`
	orderAskers := map[string]string{
		"a.ts": `import type { Order } from "./file.js";
export const f = (o: Order) => ({ shown: o.cancelled ? 1 : 2 });`,
		"b.ts": `import type { Order } from "./file.js";
export const f = (o: Order) => [o.cancelled === undefined];`,
	}
	one := rule_tester.OptionsFromJSON[any](`{"max": 1}`)
	with := func(extra map[string]string, base map[string]string) map[string]string {
		merged := map[string]string{}
		for name, code := range base {
			merged[name] = code
		}
		for name, code := range extra {
			merged[name] = code
		}
		return merged
	}
	items := `export type StockItem = { kind: "stock"; name: string };
export type Note = { kind: "note"; body: string };
export type Item = StockItem | Note;`
	itemAskers := map[string]string{
		"a.ts": `import type { Item } from "./file.js"; export const f = (i: Item) => ({ shown: i.kind === "stock" ? 1 : 2 });`,
		"b.ts": `import type { Item } from "./file.js"; export const f = (i: Item) => [i.kind !== "stock"];`,
	}
	media := `export type Image = { kind: "image"; file?: { at: number } };
export type Video = { kind: "video"; file?: { at: number } };
export type Media = Image | Video;`
	mediaAskers := map[string]string{
		"a.ts": `import type { Media } from "./file.js"; export const f = (m: Media) => ({ shown: m.file !== undefined });`,
		"b.ts": `import type { Media } from "./file.js"; export const f = (m: Media) => [m.file === undefined];`,
	}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: header + `export const f = (f: Folder) => f.isDeleted() || f.isDraft();`, Files: files},
			// The same split written with other values is another fact: a new content type would make them disagree.
			{Code: header + `export const f = (f: Folder) => f.contentType === "audio";`, Files: files},
			// A chain on one property is one test: draft or final is a question no predicate answers.
			{Code: header + `export const f = (f: Folder) => f.kind === "draft" || f.kind === "final";`, Files: files},
			{Code: header + `export const f = (f: Folder) => f.kind !== "draft" && f.kind !== "final";`, Files: files},
			// No predicate for this value, and a read is not a predicate.
			{Code: header + `export const f = (f: Folder) => f.kind === "review";`, Files: files},
			{Code: header + `export const f = (f: Folder) => (f.note ? 1 : 2);`, Files: files},
			// A truthiness test also removes "", which hasLabel does not.
			{Code: header + `export const f = (f: Folder) => (f.label ? 1 : 2);`, Files: files},
			// The test narrows the value for a later read.
			{Code: header + `export function f(f: Folder) { if (f.deleted === undefined) return 0; return f.deleted.at; }`, Files: files},
			// The owner and its subclasses read their own fields.
			{Code: `export class Folder {
  deleted?: { at: number };
  isDeleted() { return Boolean(this.deleted); }
  same(other: Folder) { return Boolean(other.deleted) === this.isDeleted(); }
  check() { return this.deleted !== undefined; }
}`},
			{Code: header + `export class Sub extends Folder { same(other: Folder) { return other.kind === "draft"; } }`, Files: files},
			// Helper functions are not predicates of the type.
			{Code: `type Plain = { kind: "a" | "b" };
export const isA = (p: Plain) => p.kind === "a";
export const g = (p: Plain) => p.kind === "a";`},
			// A union receiver loses its narrowing with a boolean predicate.
			{Code: strings.Replace(shapes, "%s", "", 1)},
			{Code: header + `export function f(f: Folder) { switch (f.kind) { case "draft": return 1; default: return 2; } }`, Files: files},
			{Code: header + `export const f = (f: any) => f.deleted !== undefined;`, Files: files},
			{Code: `import { Folder } from "./missing.js";
export const f = (f: Folder) => f.deleted !== undefined;`},
			// Helpers that are not exported or take more parameters are not predicates.
			{Code: use + `export const f = (item: Item) => (item.kind === "text" ? 1 : 2);`, Files: helped},
			// A parenthesized string is a statement before the return, not a directive.
			{Code: image, Files: directive(`("worklet");`)},
			// A module defining the same fact is a stand-in owner too.
			{Code: use + `export const isStock = (item: Item) => item.kind === "stock";`, Files: helped},
			// The helper's own file and the file declaring the property are owners.
			{Code: helpers("Item") + "\nexport const g = (item: Item) => item.kind === \"stock\";", Files: map[string]string{"items.ts": items}},
			{Code: items + "\nexport const g = (item: Item) => item.kind === \"stock\";", Files: map[string]string{"helpers.ts": helpers("Item")}, FileName: "items.ts"},
			// Only the helper's parameter type differs from the reported helper case.
			{Code: nonStock, Files: map[string]string{"items.ts": items, "helpers.ts": helpers("Wide")}},
			// Only the imported type differs from the reported comparison.
			{Code: compared, Files: map[string]string{"folder.ts": plain}},
			// A call can reach an override that answers another question, through a union member or a subclass.
			{Code: overridden},
			// An unanswered fact needs more than max files outside the owner: two files are within the default of 4.
			{Code: message, Files: askers},
			// The owner, or a helper module, already answers it: the bypassed case reports each file instead.
			{Code: strings.Replace(order, "};\n}", "};\n  isCancelled() { return this.cancelled !== undefined; }\n}", 1), Files: orderAskers, Options: one},
			{Code: message, Files: with(map[string]string{"helpers.ts": `import type { Leaflet } from "./file.js";
export const isPosted = (m: Leaflet) => m.postedAt !== undefined;`}, askers), Options: one},
			// A boolean property is its own answer.
			{Code: message, Files: map[string]string{"a.ts": asked("!!m.archived"), "b.ts": asked("(m.archived ? 1 : 2)")}, Options: one},
			// Tests in the owner's module do not count, and different facts do not add up.
			{Code: message + "\nexport const g = (m: Leaflet) => !!m.postedAt;", Files: map[string]string{"a.ts": asked("!!m.postedAt")}, Options: one},
			{Code: message, Files: map[string]string{"a.ts": asked(`m.status === "draft"`), "b.ts": asked(`m.status === "failed"`)}, Options: one},
			// A field destructured from a parameter has no expression naming its object to call the predicate on.
			{Code: header + `export const f = ({ kind }: Folder) => kind === "final";`, Files: files},
			// Tests on any are not facts about the type.
			{Code: message, Files: map[string]string{"a.ts": `export const f = (m: any) => [!!m.postedAt];`, "b.ts": `export const f = (m: any) => [!!m.postedAt];`}, Options: one},
		},
		[]rule_tester.InvalidTestCase{
			invalid(truthy, "folder.deleted", 1),
			invalid(filter, "!f.deleted", 1),
			invalid(compared, "f.deleted !== undefined", 1),
			invalid(boolean, "Boolean(f.deleted)", 1),
			invalid(values, `f.kind !== "draft"`, 1),
			invalid(unnarrowed, "f.deleted !== undefined", 1),
			invalid(arrow, `f.kind === "final"`, 1),
			invalid(getter, "f.label !== undefined", 1),
			invalid(chain, `f.contentType === "image" || f.contentType === "video"`, 1),
			invalid(alias, `entry.kind === "draft"`, 1),
			invalid(includes, `["image", "video"].includes(f.contentType)`, 1),
			invalid(generic, `f.kind === "draft"`, 1),
			invalid(destructured, `kind === "final"`, 1),
			invalid(optional, "f?.deleted", 1),
			invalid(missingMemo, "f?.memo !== undefined", 1),
			invalid(nested, `f.contentType === "image" || f.contentType === "video"`, 1),
			invalid(negatedChain, `f.contentType !== "image" && f.contentType !== "video"`, 1),
			invalid(guarded, `f.kind === "draft"`, 1),
			{Code: predicate, Errors: []rule_tester.InvalidTestCaseError{at(predicate, `s.kind === "circle"`, 2)}},
			{Code: nonStock, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(nonStock, `item.kind === "stock"`, 1)}},
			{Code: text, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(text, `m.type === "text"`, 1)}},
			{Code: taken, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(taken, `seat.kind === "taken"`, 1)}},
			{Code: done, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(done, "!task.finishedAt", 1)}},
			{Code: maybeSeat, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(maybeSeat, `seat?.kind === "taken"`, 1)}},
			{Code: maybeText, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(maybeText, `m?.type === "text"`, 1)}},
			{Code: maybeNotText, Files: helped, Errors: []rule_tester.InvalidTestCaseError{at(maybeNotText, `m?.type !== "text"`, 1)}},
			{Code: values, Files: files, Options: rule_tester.OptionsFromJSON[any](`{}`), Errors: []rule_tester.InvalidTestCaseError{at(values, `f.kind !== "draft"`, 1)}},
			{Code: instantiated, Errors: []rule_tester.InvalidTestCaseError{at(instantiated, "box.value", 2)}},
			invalid(stored, "f?.deleted !== undefined", 1),
			{Code: message, Files: askers, Options: one, Errors: []rule_tester.InvalidTestCaseError{unanswered(message, "postedAt")}},
			{Code: message, Files: statuses, Options: one, Errors: []rule_tester.InvalidTestCaseError{unanswered(message, "status")}},
			{Code: order, Files: orderAskers, Options: one, Errors: []rule_tester.InvalidTestCaseError{unanswered(order, "cancelled")}},
			// A test of a union's tag is a fact about the member it names, not about the others.
			{Code: items, Files: itemAskers, Options: one, Errors: []rule_tester.InvalidTestCaseError{unanswered(items, "kind")}},
			// Tests through a union ask about each member's declaration: the fact is reported once.
			{Code: media, Files: mediaAskers, Options: one, Errors: []rule_tester.InvalidTestCaseError{unanswered(media, "file")}},
			// A directive prologue does not count as part of the helper's body.
			{Code: image, Files: directive(`"worklet";`), Errors: []rule_tester.InvalidTestCaseError{at(image, `m.type === "image"`, 1)}},
		},
	)
}
