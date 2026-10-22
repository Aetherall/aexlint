# no-deferred-entry

Reports an entry that a local collection only holds back for the one loop over it. The entry is complete when it is stored, and nothing else reads it before the loop takes it, so the collection changes nothing but when the loop's work runs. The reader has to find that loop to learn what happens to each entry, and every entry stays alive until then. Doing the loop's work where the entry is made, or yielding the entry from a generator the loop consumes, gives the loop the same values.

```json
{ "rules": { "aexlint/no-deferred-entry": "warn" } }
```

No options.

## Example

Reported on each `shifts.push(…)`:

```ts
const shifts: Shift[] = [];
if (overtime) {
  shifts.push(new Shift(crew, 12));
} else {
  shifts.push(new Shift(crew, 8));
  if (double) shifts.push(new Shift(crew, 4));
}
for (const shift of shifts) out.push(shift.describe());
```

The branches only decide which shifts exist. A generator says so, and the loop consumes it:

```ts
function* shifts(crew: string, overtime: boolean, double: boolean): Generator<Shift> {
  if (overtime) {
    yield new Shift(crew, 12);
    return;
  }
  yield new Shift(crew, 8);
  if (double) yield new Shift(crew, 4);
}
```

Accepted when the collection does more than defer:

- an entry is looked up again, as in get-or-create, or its size is read;
- a second loop reads it;
- code between the store and the loop reads data the loop changes, as `log.push(String(log.length))` before a loop pushing into `log`: moving the loop's work would change what that read sees;
- code the model does not see runs on one side while the other side reads or changes data such code may reach, as a declared `audit(name)` call after each store.

Only a dependency's type can make the difference. With `class Crate { constructor(readonly label: string) {} }`, `crates.push(new Crate(label))` before a loop over `crates` is reported. With `declare class Crate { constructor(label: string); }`, it is not: constructing a `Crate` runs code the model does not see, and the loop reads data such code may reach.

## What counts

- The collection is a [buffer](../../docs/design/decision-model.md): a local array, `Map`, `Set` or object created empty and used only as a container, whose only read as a whole is one `for-of` loop or one `forEach` call, its drain, after every entry is stored.
- The entry is settled: no lookup, check or removal reaches it, every way from the store reaches the drain, and no other entry may replace it. Its value is a primitive, a new object or a constant, or else nothing between the store and the drain changes other data or runs code the model does not see.
- Nothing orders the deferral: no read or effect between the store and the drain depends on running before the drain, and no code the model does not see is exposed on either side.
- The entry's type does not matter: an entry of `any` is reported like any other, and an unresolved construction counts as code the model does not see.

## Reporting and limits

Reports `deferredEntry` on each store or append of a settled entry, naming the collection and the line of its loop. No fixes.

Buffers are read within one function. Settled entries whose deferral is ordered or exposed are left alone, though they may still be worth handing over: [no-split-result](../no-split-result/README.md) suggests it when a decision's branch makes up their keys.
