# no-lost-write

Reports a write to a field or array element that another write replaces before anything can read it. The written value is never seen, so the write does nothing. Either it is a leftover, or the code meant to read the value in between and doesn't.

```json
{ "rules": { "aexlint/no-lost-write": "warn" } }
```

No options.

## Example

Reported on `this.level = 0` and on `Gauge.current = undefined`:

```ts
reset(next: number): void {
  this.level = 0;
  this.label = `gauge-${next}`;
  this.level = next;
}
claim(): void {
  Gauge.current = undefined;
  Gauge.current = this;
}
```

Accepted when the value can be seen before it is replaced:

- something reads it in between;
- a call the model can't see may read it, such as `notify()` or an `await` (unless the data is private to the function);
- a way out skips the second write;
- the property has a setter, which runs code.

Only a dependency's type can make the difference. With `interface Dial { angle: number }`, `dial.angle = 0; dial.angle = next;` is reported. With `get angle()` and `set angle(value)`, it is not.

## What counts

- A lost write is an [overwritten store](../../docs/design/decision-model.md) of a declared property, not through a setter, or of an array or tuple element. The replacing write is on every way from it, under the same field, or under a key from the same origins.
- A store into a map, set or object used as a map counts only when that container is local to the function and read only by key: elsewhere the first store still fixes where its key comes in iteration order. Placeholders, which still grow their collection, and properties of `any`, `unknown` or unresolved types are skipped.
- A write that a decision's branch makes only for the code after the decision is left to [no-split-result](../no-split-result/README.md), which reports it as a fabricated outcome.

## Reporting and limits

Reports `lostWrite` on the lost write, naming the write that replaces it. No fixes.

Code the model does not see is taken to read any data it may reach, unless that data is private to the function. Keys are compared by where they come from, not by value. A property its type declares without accessors is taken as plain, even when an implementation stores it through a setter, as a library's observable values may.
