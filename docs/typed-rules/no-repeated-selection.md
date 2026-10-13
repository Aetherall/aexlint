# no-repeated-selection

Report a handler that repeats a test its sibling selector already made.

**Status:** experimental. Implements the [repeated selection design](../design/repeated-selection.md) on the [decision model](../design/decision-model.md). Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-repeated-selection": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Reported code

An API asks "is this value for you?" through a selector, then calls a handler with the same value. The selector returns `boolean`, so its answer is lost and the handler decides again:

```ts
type TextItem = { kind: "text"; text: string };
type StockItem = { kind: "stock"; sku: "alpha" | "beta" };
type Item = TextItem | StockItem;

registerCatalog({
  matches: (item: Item) => item.kind === "stock" && item.sku === "alpha",
  render: (item: Item) => (item.kind === "stock" ? renderStock(item) : undefined),
});
```

`render` declares that it handles any item but only handles stock items. To understand it, a reader must know that the API calls `matches` first. Changing the selection means changing every copy.

## What counts

- A **selector** is a function member of an object literal or class whose whole result is a boolean expression, and which tests at least one finite literal value on its first parameter, such as `item.kind === "stock"`. A type predicate (`item is StockItem`) also counts.
- A **handler** is another function member of the same object or class whose first parameter has the same type.
- The handler is reported when its entry guard narrows that parameter, and a test in the selector on the same property accepts only variants the guard keeps. Variants and property identity come from the checker, as in [no-repeated-value-group](no-repeated-value-group.md#what-counts).
- An entry guard is one of the handler's [entry tests](../design/decision-model.md#boundaries): a first statement `if (test) <exit>` whose exit does not use the value while the rest does, a whole body `test ? work : fallback` or `test && work`, or a single `if (test) work`. Re-reading the tested property itself is not using the value. See [rejection](../design/decision-model.md#rejection).
- A handler that only forwards its parameter to another project function, such as `render: (item) => this.renderCall(item)` or an imported function, is examined through that function's entry guard. The report is on the forwarding call.

## Accepted code

Reuse a positive-only selection helper that returns the selected value:

```ts
function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "stock" && item.sku === "alpha" ? item : undefined;
}

registerCatalog({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? renderStock(selected) : undefined;
  },
});
```

The helper promises only that its returned value is a `StockItem`. Returning `undefined` does not claim the input cannot be a stock item: a stock item with sku `"beta"` is correctly rejected by this policy. No assertion or type predicate is needed.

A type predicate is appropriate only for an exact two-sided partition: `item.kind === "stock"` can establish `item is StockItem` for a closed `Item = TextItem | StockItem` union. Adding `item.sku === "alpha"` cannot retain that broad predicate target because it rejects other permitted stock items. See [no-overbroad-type-predicate](no-overbroad-type-predicate.md).

A local adapter whose members both call one shared guard or selected-value helper, an API whose sound exact selector is a type predicate with a handler typed with the narrowed type, a handler that does not test its parameter, and a handler testing something its selector does not test are all accepted. Event handlers and list callbacks have no sibling selector.

Changing only an imported type can change the result: widening `StockItem.kind` from `"stock"` to `string` removes the finite vocabulary, so nothing is reported.

## Reporting scope

Reports `repeatedSelection` once per narrowed handler parameter, on the repeated test or on the forwarding call. The message names the selector. The help recommends carrying the narrowed selected value into the handler or reusing a shared selection helper. For a project-owned contextual API type, it names that type as the place to carry the selected value. For a dependency or declaration-file API type, it recommends a local selected-value adapter or shared helper. Every case limits type-predicate advice to exact two-sided partitions, where false excludes every value of the target too.

## Fixes

None. Each accepted design changes types or structure beyond the reported expression.

## Limitations

- Selectors and handlers are related only inside one object literal or class. A selector registered separately from its handler is not connected.
- Calls in a selector to project functions with a body, including helpers imported from other files of the program, are expanded into their tests, up to three calls deep. Calls in the handler are reuse and never count as a repeated test. Declared functions without a body are not expanded.
- Tests on open `string` values have no finite vocabulary and are not compared.
- The detector does not verify runtime call order or selector purity. Findings do not establish that removing a check preserves behavior or improves readability.
