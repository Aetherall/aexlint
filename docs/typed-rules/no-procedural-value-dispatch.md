# no-procedural-value-dispatch

Reports branching on a literal union where a branch has its own control flow. That branch is a procedure for its values, written inline in a function with another job; it is easier to name, test and change on its own.

```json
{ "rules": { "aexlint-typed/no-procedural-value-dispatch": "warn" } }
```

No options.

## Example

Reported on `order.delivery`:

```ts
orders.map(async (order) => {
  if (order.delivery === "parcel") return { order, label: await weigh(order) };
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

Accepted: branches that return a value, make one call, or run a straight sequence of effects.

## What counts

- A dispatch is a `switch` on a vocabulary, or an `if`/`else if` chain, a run of `if`s, or a ternary chain testing the same subject. A test may combine the subject with other conditions if it still restricts the subject's values. A single tested `if` needs an `else`.
- A branch is procedural when it contains an `if`, `switch`, ternary, loop or `try`, including in inline callbacks. `&&`, `||` and `??` are not control flow.
- `default`, trailing `else` and code after an `if` run are not examined.
- Subjects are compared as values: `const { type } = a` makes `type` the same subject as `a.type`.

## Reporting and limits

Reports `proceduralBranch` once per dispatch, on its first subject; help lists the procedural values. No fixes.

Any nested decision counts, even a short ternary. Lookup tables, handler maps and `switch (true)` are not recognized. Deserializers translating every variant may need an override.
