package decision_model

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const catalog = `export type TextItem = { kind: "text" | "note"; text: string };
export type StockItem = { kind: "stock"; sku: string };
export type Item = TextItem | StockItem;
export type CatalogView = {
  matches(item: Item): boolean;
  render(item: Item): string | undefined;
};
export declare function registerView(view: CatalogView): void;`

func at(id, code, snippet string, occurrence int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range occurrence {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: id, Line: strings.Count(before, "\n") + 1, Column: column, EndColumn: column + len(snippet)}
}

func TestDecisionModel(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"catalog.ts": catalog}
	repeated := `import { registerView } from "./catalog.js";
registerView({
  matches: (item) => item.kind === "stock" && item.sku === "alpha",
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`
	passthrough := `import type { Item, CatalogView } from "./catalog.js";
export class CatalogRenderer {
  view: CatalogView = {
    matches: (item) => item.kind === "stock" && item.sku === "beta",
    render: (item) => this.renderStock(item),
  };
  private renderStock(item: Item) {
    if (item.kind !== "stock") return undefined;
    return item.sku;
  }
}`
	event := `type Item = { kind: "text" | "note"; text: string };
declare function on(listener: (item: Item) => void): void;
declare function finish(text: string): void;
on((item) => { if (item.kind === "note") finish(item.text); });`
	conflict := `type Node = { status: "running" | "failed" | "done" };
declare function paint(color: "accent" | "error" | "dim" | "success", text: string): string;
export function header(node: Node) {
  const color = node.status === "failed" ? "error" : node.status === "running" ? "accent" : "dim";
  return paint(color, "header");
}
export function row(node: Node) {
  const color = node.status === "running" ? "accent" : node.status === "failed" ? "error" : "success";
  const icon = node.status === "running" ? "*" : node.status === "failed" ? "x" : "ok";
  return paint(color, icon);
}`
	returned := `type Node = { status: "running" | "failed" | "done" };
declare function paint(color: "accent" | "error" | "dim" | "success", text: string): string;
function statusColor(node: Node) {
  switch (node.status) {
    case "running": return "accent";
    case "failed": return "error";
    default: return "dim";
  }
}
function rowColor(node: Node) {
  return node.status === "running" ? "accent" : node.status === "failed" ? "error" : "success";
}
export function render(node: Node) {
  return paint(statusColor(node), "header") + paint(rowColor(node), "row");
}`
	helper := `import { registerView, type Item, type StockItem } from "./catalog.js";
const isStock = (item: Item): item is StockItem => item.kind === "stock";
registerView({
  matches: (item) => isStock(item) && item.sku === "alpha",
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`
	disjunction := `type Event = { name: "created" | "seeded" | "deleted" | "renamed" };
declare function describe(event: Event): number;
function isInitial(event: Event) { return event.name === "created" || event.name === "seeded"; }
export const handler = {
  accepts: (event: Event) => isInitial(event) || event.name === "deleted",
  apply(event: Event) {
    if (event.name === "renamed") return undefined;
    return describe(event);
  },
};`
	narrowed := `type Reorder = { kind: "reorder"; key: string };
type Insert = { kind: "insert"; target: "cell" | "row"; template: string };
type Drop = Reorder | Insert;
declare function paint(color: "accent" | "dim", text: string): string;
export function first(item: Drop) {
  if (item.kind !== "insert") return "";
  return paint(item.target === "cell" ? "accent" : "dim", item.template);
}
export function second(item: Insert) {
  return paint(item.target === "cell" ? "dim" : "accent", item.template);
}`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: `type Node = { status: "running" | "failed" | "done"; kind: "parcel" | "stock" };
export function pending(node: Node) {
  if (node.status === "running") return node.kind === "parcel" ? "parcel" : "stock";
  return "none";
}`},
			{Code: `type Item = { kind: "text" | "note" };
export function frame(item: Item) {
  if (item.kind === "text") return { rail: "|" };
  return undefined;
}`},
			{Code: `import type { Item, StockItem } from "./catalog.js";
import { registerView } from "./catalog.js";
const isStock = (item: Item): item is StockItem => item.kind === "stock";
registerView({ matches: (item) => item.kind === "note", render: () => "suggested" });
registerView({ matches: isStock, render: (item) => (isStock(item) ? item.sku : undefined) });
registerView({ matches: (item) => isStock(item) && item.sku === "alpha", render: (item) => (isStock(item) ? item.sku : undefined) });`, Files: files},
			{Code: `type Event = { name: "created" } | { name: "deleted" } | { name: "renamed"; to: string };
export class Projection {
  apply(event: Extract<Event, { name: "created" | "deleted" }>) { return event.name; }
}`},
			{Code: `declare function t(key: string): string;
type Props = { title: string; description: string };
declare function show(props: Props): void;
export function sheet(kind: "team" | "project") {
  show({
    title: t(kind === "team" ? "title.team" : "title.project"),
    description: t(kind === "team" ? "description.team" : "description.project"),
  });
}`},
			{Code: `export function marker(env: "prod" | "dev", kind: "photo" | "plain") {
  switch (env) {
    case "prod": return kind === "photo" ? "p1" : "p2";
    case "dev": return kind === "photo" ? "d1" : "d2";
  }
}`},
			{Code: `declare function paint(color: "warning" | "muted" | "accent", text: string): string;
export function status(working: boolean, mode: "queue" | "immediate") {
  if (!working) return paint(mode === "queue" ? "warning" : "muted", "idle");
  return paint(mode === "queue" ? "warning" : "accent", "busy");
}`},
			{Code: `declare function memo<T>(f: () => T): T;
export function sizes(size: "S" | "M" | "L") {
  const format = memo(() => { switch (size) { case "S": case "M": return "narrow"; default: return "long"; } });
  const text = memo(() => { switch (size) { case "S": return "MINI"; default: return "BODY"; } });
  return [format, text];
}`},
			{Code: `declare function paint(color: "accent" | "dim", text: string): string;
declare function label(text: string): string;
function color(mode: "on" | "off") { return mode === "on" ? "accent" : "dim"; }
function text(mode: "on" | "off") { return mode === "on" ? "enabled" : "disabled"; }
export function show(mode: "on" | "off") { return paint(color(mode), label(text(mode))); }`},
			{Code: `export function f(item: any) { if (item.kind !== "stock") return undefined; return item.name; }`},
			{Code: `export function f(item: Missing) { if (item.kind !== "stock") return undefined; return item.name; }`},
			{Code: `export function f(item: unknown) { return item; }`},
			{Code: `export function f(item: never) { return item; }`},
			{Code: repeated, Files: map[string]string{"catalog.ts": strings.Replace(catalog, `kind: "stock"`, "kind: string", 1)}},
		},
		[]rule_tester.InvalidTestCase{
			{Code: repeated, Files: files, Errors: []rule_tester.InvalidTestCaseError{
				at("ownNarrowing", repeated, `item.kind === "stock"`, 1),
				at("selectorNarrowing", repeated, `item.kind === "stock"`, 2),
			}},
			{Code: passthrough, Files: files, Errors: []rule_tester.InvalidTestCaseError{
				at("ownNarrowing", passthrough, `item.kind === "stock"`, 1),
				at("selectorNarrowing", passthrough, `this.renderStock(item)`, 1),
				at("ownNarrowing", passthrough, `item.kind !== "stock"`, 1),
			}},
			{Code: event, Errors: []rule_tester.InvalidTestCaseError{at("ownNarrowing", event, `item.kind === "note"`, 1)}},
			{Code: conflict, Errors: []rule_tester.InvalidTestCaseError{at("conflict", conflict, `node.status`, 1)}},
			{Code: returned, Errors: []rule_tester.InvalidTestCaseError{at("conflict", returned, `node.status`, 1)}},
			{Code: helper, Files: files, Errors: []rule_tester.InvalidTestCaseError{at("selectorNarrowing", helper, `item.kind === "stock"`, 2)}},
			{Code: narrowed, Errors: []rule_tester.InvalidTestCaseError{
				at("ownNarrowing", narrowed, `item.kind !== "insert"`, 1),
				at("conflict", narrowed, `item.target`, 1),
			}},
			{Code: disjunction, Errors: []rule_tester.InvalidTestCaseError{at("selectorNarrowing", disjunction, `event.name === "renamed"`, 1)}},
		},
	)
}
