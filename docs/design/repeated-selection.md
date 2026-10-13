# Repeated selection

**Status: design specification, implemented experimentally as [no-repeated-selection](../typed-rules/no-repeated-selection.md) on the [decision model](decision-model.md).**

## Problem

Some APIs split one decision into two parts: a selector that answers "is this value for me?" and a handler that does the work. When the selector returns `boolean`, its answer is lost. The handler receives the whole declared type and must decide again.

```ts
type TextItem = { kind: "text"; text: string };
type StockItem = { kind: "stock"; sku: "alpha" | "beta" };
type Item = TextItem | StockItem;

registerCatalog({
  matches: (item: Item) => item.kind === "stock" && item.sku === "alpha",
  render: (item: Item, context) => (item.kind === "stock" ? renderStock(item, context) : undefined),
});
```

`render` declares that it handles any item, but it only handles stock items. The decision is written twice: once in `matches`, once in `render`. To understand `render`, a reader must know that the API calls `matches` first. A function that depends on its callers to describe itself is the problem.

The failing branch in `render` exists only because its type is wider than its real input. Changing the selection means finding and changing every copy.

## Principle

**A selection decision is written once, by its selector. Code gated by that selector reuses its answer instead of writing the decision again.**

This is a property of the declaring code, not of its callers. The detector relates a selector and a sibling handler in one object or class; it does not verify the API's runtime call order.

## Accepted designs

The rule accepts any design in which the decision is written once.

### A selected-value interface

When the project owns the API, a selection can carry the narrowed value instead of returning a boolean:

```ts
type SelectedCatalogHandler<T extends Item> = {
  select(item: Item): T | undefined;
  render(item: T, context: RenderContext): string | undefined;
};
```

This supports positive-only policies: `undefined` means no selection, not that the original value lies outside `T`. An API using `matches(item): item is T` instead requires an exact two-sided partition. For a closed item union, `item.kind === "stock"` can identify all `StockItem` values; `item.kind === "stock" && item.sku === "alpha"` cannot promise `item is StockItem`, because stock items with sku `"beta"` fail. Do not turn compound selection into a broad iff predicate.

### A local adapter

When the API cannot change, a local adapter narrows once and gives handlers the narrowed type:

```ts
const isStock = (item: Item): item is StockItem => item.kind === "stock";

function stockHandler(handler: {
  matches(item: StockItem): boolean;
  render(item: StockItem, context: RenderContext): string | undefined;
}): CatalogHandler {
  return {
    matches: (item) => isStock(item) && handler.matches(item),
    render: (item, context) => (isStock(item) ? handler.render(item, context) : undefined),
  };
}
```

Both members call the same exact guard, `isStock`. The sku policy remains boolean in `handler.matches`, separate from the two-sided kind partition. The kind test is written once, in that guard. Calling it twice is reuse, not repetition.

### Reusing a selected-value helper

Both members can reuse a true-only selection without claiming false excludes every stock item:

```ts
function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "stock" && item.sku === "alpha" ? item : undefined;
}

registerCatalog({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item, context) => {
    const selected = selectAlpha(item);
    return selected ? renderStock(selected, context) : undefined;
  },
});
```

The returned value carries `StockItem` typing without an assertion. Both members call the same helper; calling it twice is reuse, not repetition. The rule does not establish purity or permission to remove a runtime check.

## Candidate diagnostic

> This repeats a decision already made by `matches`.

Help depends on where the API type is declared:

- In the project: carry the narrowed selected value through the API into the handler, or reuse a shared selection helper.
- In a dependency: use a local selected-value adapter, or reuse a shared selection helper.

In either case, recommend a type predicate only for an exact two-sided partition. The detector finds repetition; it does not prove the selector's false-side contract.

Span: the repeated test in the handler, or a forwarding call whose target repeats the test. Report once per narrowed handler parameter.

No automatic fix: each accepted fix changes types or structure beyond the reported expression.

## Detection specification

1. Find an object literal or class whose members include a **selector**: a function whose whole result is a boolean expression testing finite literal values on its first parameter, or a declared type predicate.
2. Find a **sibling handler** in the same object or class whose first parameter has the same declared type.
3. Read the selector's tests through the checker and the [decision model](decision-model.md#predicates), including supported calls to project functions with bodies, up to three calls deep.
4. Report when a handler's entry guard narrows its parameter and the selector's claim on the same property implies what that guard keeps.

Property identity and literal values come from the checker, not spelling. Variant tags and plain values count alike: `sku !== "beta"` repeats `sku === "alpha"` when the declared sku domain is exactly `"alpha" | "beta"`.

Selectors and handlers can be arrow functions, function expressions, methods, or references to functions. A handler that only forwards its parameter to a project function, such as `render: (item, context) => renderStock(item, context)`, is examined through that function's entry condition.

## Not reported

- A selector that returns a type predicate, with a handler typed with the narrowed type.
- A handler that calls the selector as its guard.
- A handler whose entry test is not implied by its sibling selector's tests. That is the handler's own decision.
- Functions with no sibling selector, such as event handlers and list callbacks (`.filter`, `.find`, `.flatMap`).
- A selector and a handler that both call the same guard function, as in the local adapter above. Handler guard calls are reuse.
- A selector whose tests read state outside its parameter only (for example `() => enabled`). It has no tests on the parameter to repeat.

## Dependencies

API types declared in dependencies are not skipped. The repeated decision and the accepted fixes are both in the project's own code.

## Limitations

- Only selectors and handlers in the same object or class are related. A selector registered separately from its handler is not connected.
- Declared functions without a body are not expanded. Calls in handler guards are reuse, not repeated tests.
- Tests on open `string` values have no finite vocabulary and are not compared. Equivalent tests outside the supported forms, such as `in` checks against a tag comparison, are not matched.
- Gates inside libraries, such as a filter followed by a map or an acceptance callback followed by a drop handler, are not related to each other.
- Repetition across different objects, for example two catalog handlers making the same selection, is a separate concern.
- Diagnostic mechanics do not establish usefulness, completeness, or readability improvement. Review findings before adopting this experimental rule.
