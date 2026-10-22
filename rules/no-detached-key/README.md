# no-detached-key

Reports a method of a map's value that is given the key the value is stored under, when the map only holds values made for their key. The value needs its own key to do its work, but the map holds the key outside it. So every reader of the map has to pass the pair together, and a value taken out of the map can no longer do its work alone.

```json
{ "rules": { "aexlint/no-detached-key": "warn" } }
```

No options.

## Example

Reported on `cell.render(column, width)`:

```ts
const cells = new Map<number, Cell>();
for (const [column, text] of texts.entries()) {
  cells.set(column, cells.get(column) ?? new Cell(text));
}
for (const [column, cell] of cells) out.push(cell.render(column, width));
```

Accepted when one value is stored under several keys, such as one handler for several events: the handler is told which event it runs for. Also accepted when the values are numbers or other data the function did not make for the key.

Only a dependency's type can make the difference. With `class Slot { fill(name: string) { … } }`, `slot.fill(name)` over a map of new slots is reported. With `declare class Slot { fill(name: string): string; }`, it is not, because the method has no body the model can follow.

## What counts

- The map is local: a variable created empty as a global `Map`, or as an object used as a map, and used only as a container (see [the decision model](../../docs/design/decision-model.md)). Every entry stores, under a key, a new object made in the function, or the value read back from the map itself, as in get-or-create.
- The map is read by a `for-of` loop over its entries binding `[key, value]` (the map itself, its `entries()`, or `Object.entries` of the object), by a `Map`'s `forEach((value, key) => …)`, or by `forEach(([key, value]) => …)` over `Object.entries` of the object.
- In that loop or callback, a method of the value that resolves to a project function with a body is given the key itself as an argument.

## Reporting and limits

Reports `detachedKey` on each such call, naming the method, the key and the map. No fixes.

Maps stored in fields or passed in are skipped, since their contents are not known. So are arrays, whose index is their order rather than a key, and keys passed through another variable or a property.
