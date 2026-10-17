# no-repeated-selection

Reports a handler that repeats a test its sibling selector already made. The selector returns `boolean`, so its answer is lost and the handler decides again; changing the selection means changing both.

```json
{ "rules": { "aexlint-typed/no-repeated-selection": "warn" } }
```

No options.

## Example

Reported on the test in `render`:

```ts
registerCatalog({
  matches: (item: Item) => item.kind === "stock" && item.sku === "alpha",
  render: (item: Item) => (item.kind === "stock" ? renderStock(item) : undefined),
});
```

Accepted: both members share one helper that returns the selected value.

```ts
const selectAlpha = (item: Item) =>
  item.kind === "stock" && item.sku === "alpha" ? item : undefined;
registerCatalog({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? renderStock(selected) : undefined;
  },
});
```

## What counts

- A selector is a function member of an object literal or class whose result is a boolean expression or type predicate testing finite literal values of its first parameter. Project helpers it calls are expanded up to three calls deep.
- A handler is another member of the same object or class whose first parameter has the same type.
- The handler's entry guard narrows that parameter, and the selector's test on the same property accepts only values the guard keeps. An entry guard is a first `if (test) exit`, a whole body `test ? work : fallback` or `test && work`, or a single `if (test) work`.
- A handler forwarding its parameter to a project function is checked through that function, and reported on the forwarding call.

## Reporting and limits

Reports `repeatedSelection` once per handler parameter; the message names the selector. No fixes.

A selector registered apart from its handler is not connected. Open `string` tests are not compared. Call order and selector purity are not verified. A type predicate fits only an exact partition; see [no-overbroad-type-predicate](no-overbroad-type-predicate.md).
