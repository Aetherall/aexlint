# no-external-variant-behavior

Reports code outside a type's owner that dispatches on its variants and rebuilds results from their fields. That behavior belongs to the type; each outside copy must change when a variant does.

```json
{ "rules": { "aexlint-typed/no-external-variant-behavior": "warn" } }
```

No options.

## Example

Reported in a consumer module:

```ts
function transformFigure(figure: Figure) {
  switch (figure.kind) {
    case "line":
      return { value: figure.start + 1 };
    case "circle":
      return { value: figure.center + 1 };
  }
}
```

Accepted: the same code as a method or static of `Figure`, in the module declaring a type-only `Figure`, or a dispatch that only delegates (`case "line": return figure.transform();`).

## Ownership

Resolved from declarations, not names or directories:

1. A class instance belongs to its class, including through aliases and generic constraints.
2. A type merged with a runtime value belongs to that value's initializer or namespace.
3. Any other named type or interface belongs to its declaring module or namespace. Importing or re-exporting it does not transfer ownership.

Declaration-file, dependency and unresolved owners are skipped.

## What counts

- A dispatch is a `switch`, `if`/`else if` chain, `if` run or ternary chain on a property of finite literal values, with at least two arms. A `default` or `else` counts as an arm only when every test is purely on the vocabulary.
- At least one arm returns, assigns or initializes an object or array literal reading the receiver's other fields, including inside constructor or call arguments.
- A conversion's result has a destination type, from context or construction. When it is a project type with another owner, either owner may hold the dispatch, and it is reported as `externalVariantConversion`.

## Reporting and limits

Reports once per dispatch on its first tested subject: `externalVariantBehavior` names the owner and the fields used, `externalVariantConversion` reports a conversion outside both owners. No fixes.

Adapters and serializers are not recognized; exclude them with overrides. Fallthrough, rest elements, helpers hiding construction, predicates hiding tag tests and handler tables are not followed. The rule does not detect duplicated conversions.
