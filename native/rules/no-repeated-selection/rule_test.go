package no_repeated_selection

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

func repeated(code, snippet string, occurrence int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range occurrence {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "repeatedSelection", Line: strings.Count(before, "\n") + 1, Column: column, EndColumn: column + len(snippet)}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"catalog.ts": catalog}
	ternary := `import { registerView } from "./catalog.js";
registerView({
  matches: (item) => item.kind === "stock" && item.sku === "alpha",
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`
	guard := `import { registerView } from "./catalog.js";
registerView({
  matches(item) { return item.kind === "stock"; },
  render(item) {
    if (item.kind !== "stock") return undefined;
    return item.sku;
  },
});`
	forwarded := `import type { Item, CatalogView } from "./catalog.js";
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
	stored := `import { registerView, type Item } from "./catalog.js";
const renderers = {
  stock: (item: Item) => (item.kind === "stock" ? item.sku : undefined),
};
registerView({
  matches: (item) => item.kind === "stock" && item.sku === "alpha",
  render: (item) => renderers.stock(item),
});`
	logical := `import type { Item } from "./catalog.js";
export class Labels {
  matches(item: Item) { return item.kind === "stock"; }
  label(item: Item) { return item.kind === "stock" && item.sku; }
}`
	method := `import type { Item } from "./catalog.js";
export class Renderer {
  matches(item: Item) { return item.kind === "stock"; }
  render(item: Item) { return item.kind === "stock" ? item.sku : undefined; }
}`
	aliased := `import type { Item as Entry } from "./catalog.js";
type CatalogEntry = Entry;
export const view = {
  matches: (item: CatalogEntry) => item.kind === "stock",
  render: (item: CatalogEntry) => (item.kind === "stock" ? item.sku : undefined),
};`
	shorthand := `import { registerView, type Item } from "./catalog.js";
const matches = (item: Item): boolean => item.kind === "stock" && item.sku === "alpha";
registerView({ matches, render: (item) => (item.kind === "stock" ? item.sku : undefined) });`
	helper := `import { registerView, type Item, type StockItem } from "./catalog.js";
const isStock = (item: Item): item is StockItem => item.kind === "stock";
registerView({
  matches: (item) => isStock(item) && item.sku === "alpha",
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`
	stock := map[string]string{"catalog.ts": catalog, "stock.ts": `import type { Item, StockItem } from "./catalog.js";
export const isStock = (item: Item): item is StockItem => item.kind === "stock";
export function renderStock(item: Item) {
  if (item.kind !== "stock") return undefined;
  return item.sku;
}`}
	imported := `import { registerView } from "./catalog.js";
import { isStock } from "./stock.js";
registerView({
  matches: (item) => isStock(item) && item.sku === "alpha",
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`
	importedForward := `import { registerView } from "./catalog.js";
import { renderStock } from "./stock.js";
registerView({
  matches: (item) => item.kind === "stock" && item.sku === "beta",
  render: (item) => renderStock(item),
});`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: `import { registerView, type Item, type StockItem } from "./catalog.js";
function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "stock" && item.sku === "alpha" ? item : undefined;
}
registerView({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? selected.sku : undefined;
  },
});`, Files: files},
			{Code: `import { registerView, type Item, type StockItem } from "./catalog.js";
const matches = (item: Item): item is StockItem => item.kind === "stock";
registerView({ matches, render: (item) => (matches(item) ? item.sku : undefined) });`, Files: files},
			{Code: `import { registerView } from "./catalog.js";
import { selectAlpha } from "./selection.js";
registerView({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? selected.sku : undefined;
  },
});`, Files: map[string]string{"catalog.ts": catalog, "selection.ts": `import type { Item, StockItem } from "./catalog.js";
export function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "stock" && item.sku === "alpha" ? item : undefined;
}`}},
			{Code: `import { registerView, type Item, type StockItem } from "./catalog.js";
const isStock = (item: Item): item is StockItem => item.kind === "stock";
registerView({
  matches: (item) => isStock(item) && item.sku === "alpha",
  render: (item) => (isStock(item) ? item.sku : undefined),
});`, Files: files},
			{Code: `import { registerView } from "./catalog.js";
registerView({ matches: (item) => item.kind === "note", render: () => "suggested" });`, Files: files},
			{Code: `import { registerView } from "./catalog.js";
registerView({
  matches: (item) => item.kind === "stock",
  render: (item) => (item.kind === "text" ? item.text : undefined),
});`, Files: files},
			{Code: `import { registerView } from "./catalog.js";
registerView({
  matches: () => true,
  render: (item) => (item.kind === "stock" ? item.sku : undefined),
});`, Files: files},
			{Code: `type StockItem = { kind: "stock"; sku: string };
type TextItem = { kind: "text"; text: string };
type Item = StockItem | TextItem;
type Narrow = { matches(item: Item): item is StockItem; render(item: StockItem): string };
export const view: Narrow = {
  matches: (item): item is StockItem => item.kind === "stock",
  render: (item) => item.sku,
};`},
			{Code: `type Item = { kind: "text" | "note"; text: string };
declare function on(listener: (item: Item) => void): void;
declare function finish(text: string): void;
on((item) => { if (item.kind === "note") finish(item.text); });`},
			{Code: `export const view = {
  matches: (item: any) => item.kind === "stock",
  render: (item: any) => (item.kind === "stock" ? item.name : undefined),
};`},
			{Code: `export const view = {
  matches: (item: unknown) => item === "stock",
  render: (item: unknown) => item,
};`},
			{Code: `export const view = {
  matches: (item: never) => item,
  render: (item: never) => item,
};`},
			{Code: `export const view = {
  matches: (item: Missing) => item.kind === "stock",
  render: (item: Missing) => (item.kind === "stock" ? item.name : undefined),
};`},
			{Code: ternary, Files: map[string]string{"catalog.ts": strings.Replace(catalog, `kind: "stock"`, "kind: string", 1)}},
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
			// A helper declared without a body has no tests to read.
			{Code: imported, Files: map[string]string{"catalog.ts": catalog, "stock.d.ts": `import type { Item, StockItem } from "./catalog.js";
export declare const isStock: (item: Item) => item is StockItem;`}},
		},
		[]rule_tester.InvalidTestCase{
			{Code: ternary, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(ternary, `item.kind === "stock"`, 2)}},
			{Code: guard, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(guard, `item.kind !== "stock"`, 1)}},
			{Code: forwarded, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(forwarded, `this.renderStock(item)`, 1)}},
			{Code: method, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(method, `item.kind === "stock"`, 2)}},
			{Code: aliased, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(aliased, `item.kind === "stock"`, 2)}},
			{Code: shorthand, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(shorthand, `item.kind === "stock"`, 2)}},
			{Code: helper, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(helper, `item.kind === "stock"`, 2)}},
			{Code: ternary, Files: map[string]string{"catalog.d.ts": catalog}, Errors: []rule_tester.InvalidTestCaseError{repeated(ternary, `item.kind === "stock"`, 2)}},
			{Code: stored, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(stored, `renderers.stock(item)`, 1)}},
			{Code: logical, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(logical, `item.kind === "stock"`, 2)}},
			{Code: imported, Files: stock, Errors: []rule_tester.InvalidTestCaseError{repeated(imported, `item.kind === "stock"`, 1)}},
			{Code: importedForward, Files: stock, Errors: []rule_tester.InvalidTestCaseError{repeated(importedForward, `renderStock(item)`, 1)}},
		},
	)
}
