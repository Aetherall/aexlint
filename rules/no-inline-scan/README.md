# no-inline-scan

Reports a loop that emits values computed from state it carries from one pass to the next: a running column, a cursor, the current node of a walk, the batch being filled, a counter numbering what it emits. Each value depends on what earlier passes did, so a reader runs the loop in their head to know it, down to whether it emits before or after advancing the state. The loop is a sequence with its own state, written inline. A generator owning that state and yielding the values names the sequence and keeps its state with it; when the loop also does other work, that work then only sees finished values.

```json
{ "rules": { "aexlint/no-inline-scan": "warn" } }
```

No options.

## Example

Reported on `found.push(…)`: each hit's position depends on the widths before it.

```ts
let column = 0;
for (const width of widths) {
  const end = column + width;
  found.push(new Hit(column, end));
  column = end + 2;
}
```

Reported on `names.push(parent.name)`: a walk up a chain of parents.

```ts
let current = folder.parent;
while (current) {
  const parent: Folder = current;
  names.push(parent.name);
  current = parent.parent;
}
```

The generator names what the loop computes:

```ts
function* hits(widths: number[]): Generator<Hit> {
  let column = 0;
  for (const width of widths) {
    yield new Hit(column, column + width);
    column += width + 2;
  }
}
```

Accepted when no carried state reaches what is emitted: mapping each element (`out.push(width * 2)`), folding into one value (`sum += width`), numbering by the loop's own counter (`for (let index = 0; …; index++) out.push(widths[index] + index)`), keeping a count beside the output, state declared inside the loop or fixed before it, and a generator yielding from its own state.

Only a dependency's type can make the difference. With `class Sink { put(value: number) { this.values.push(value); } }`, `sink.put(offset)` in a loop advancing `offset` is reported: the call appends the value. With `declare class Sink { put(value: number): void; }`, it is not, since what the call does with the value is not known.

## What counts

- The state is a variable of the function, declared before the loop, that the loop rebinds outside its own iteration control (a `for` statement's initializer and incrementor, or the variable a `for-of` or `for-in` loop assigns), and reads other than where a statement writes it.
- An emitted value is one the loop appends or stores, through a global collection's method, an element assignment or a project function whose summary does it, into data declared outside the loop, when the value or its key reads the state: directly, or through a constant of the pass initialized from it. The value's type does not matter.
- The rule names the loop's other work: effects on other data declared outside the loop, and variables it rebinds but only reads where a statement writes them, which build up a separate result.

## Reporting and limits

Reports `inlineScan` on the first emit of each loop, naming the state, the part of it the values are computed from when the rest only decides when they come (a page's height, a guard counting requests), and where the values go. When the only state is a counter of passes, written once per pass by `++` or `+= 1` in the loop body itself with no `continue` before it, the help points to the iteration owning it (a `for` incrementor, `entries()`, or an `enumerate` generator) rather than a generator of its own. No fixes.

The rule reads one function. Values emitted inside callbacks the loop declares, and values given to code the model does not see, are not read. Loops that only advance state without emitting, such as a fold, are not reported.
