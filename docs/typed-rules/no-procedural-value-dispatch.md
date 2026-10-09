# no-procedural-value-dispatch

Report branching on a literal union where a branch contains its own control flow. Such a branch is a procedure for its values, written inline in a function with another job. It is easier to name, test and change when it stands on its own.

**Status:** experimental. Diagnostic-only; no fixes or suggestions. See [Evidence](no-repeated-value-group.md#evidence).

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-procedural-value-dispatch": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Example

Reported on `order.delivery`: the letter branch handles destination and failure cases inside a shipping loop.

```ts
orders.map(async (order) => {
  if (order.delivery === "parcel") {
    return { order, label: await weigh(order) };
  }
  if (order.delivery === "letter" && !abroad) {
    try {
      return { order, label: await insure(order) };
    } catch (error) {
      logger.error(error);
    }
  }
  return { order, label: undefined };
});
```

Accepted when each branch returns a value, renders an element, makes one call or runs a straight sequence of effects:

```ts
switch (event) {
  case "paid":
    record(order);
    notify(order);
    break;
  case "refunded":
    notify(order);
    return;
}
```

## What counts

- Vocabularies follow [no-repeated-value-group](no-repeated-value-group.md#what-counts).
- A dispatch is a `switch` on a vocabulary, an `if`/`else if` chain whose tests all test the same subject of one vocabulary, consecutive `if` statements without `else` testing the same subject, or a chain of conditional expressions doing so. Tests may combine the subject with other conditions. An `if` with one tested branch counts only when it has an `else`.
- A branch is procedural when it contains an `if`, `switch`, conditional expression, loop or `try`, including inside arrow functions and function expressions written in the branch. Nested function declarations and methods are not searched.
- `default` branches, `else` branches after the last test, and statements after an `if` sequence are not examined.
- Where the vocabulary is declared does not matter.

## Reporting scope

Reports `proceduralBranch` once per dispatch, on the subject of its first test. Help lists the values of the procedural branches.

## Boundaries

Deserializers that translate every variant may contain per-variant logic. Turn the rule off for them with Oxlint overrides, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries).

## Limitations

- Any nested decision makes a branch procedural, including a short ternary or optional default. The rule does not measure how much work a branch does.
- Logical `&&`, `||` and `??` inside a branch are not control flow for this rule.
- Dispatches written as object lookups, maps of handlers, `switch (true)` or tests of different subjects are not recognized.
- Subjects are compared by their text, so `a.type` and an alias of it are different subjects.
