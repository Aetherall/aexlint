# no-positional-separator

Reports a separator a loop places by the pass's position instead of between the pieces it emits: emitted on every pass but the first (`if (index > 0)`) or every pass but the last (`if (index < items.length - 1)`), or appended with every piece and cut off by its length after the loop. A separator belongs between two pieces, but the position of a pass only says how many passes ran, not how many pieces were emitted. The two agree only while every pass emits a piece, which nothing states: a pass that ends early leaves the separator at the start, at the end or twice in a row, and a cut after a loop that never ran removes text written before it.

```json
{ "rules": { "aexlint/no-positional-separator": "warn" } }
```

No options.

## Example

Reported on `if (windowIndex > 0) …`: when the first window is empty, the evidence starts with `"…"`.

```ts
windows.forEach((block, windowIndex) => {
  if (block.length === 0) return;
  if (windowIndex > 0) evidence.push("…");
  evidence.push(...block);
});
```

Reported on `if (index > 0) this.push(", ")`, which is right only while every pass pushes an argument:

```ts
for (const [index, arg] of args.entries()) {
  if (index > 0) this.push(", ");
  this.term(arg);
}
```

Reported on `text.slice(0, -2)`, which cuts two characters of `content` when there are no cells:

```ts
let text = content;
for (const cell of cells) text += cell + "  ";
return text.slice(0, -2);
```

Accepted: joining the pieces, and tracking whether a piece was emitted, which places the separator by adjacency.

```ts
return blocks
  .filter((block) => block.length > 0)
  .map((block) => block.join("\n"))
  .join("\n…\n");

let emitted = false;
for (const name of names) {
  if (!name) continue;
  if (emitted) out += ", ";
  out += name;
  emitted = true;
}
```

Also accepted when what the guard emits depends on the pass (`if (index > 0) out += cell.separator`), when it marks one end rather than skipping it (`if (index === 0) out += "Names: "`), when it goes somewhere the pieces do not, and when a cut is longer than the literal text every piece ends with, runs inside the loop, or follows a write after the loop.

## What counts

- A pass has an index: the counter a `for` statement declares, the first element of `[index, value]` in a `for-of` loop over `entries()`, or the second parameter of a callback given to `forEach`, `map` or `flatMap`.
- The guard is an `if` without `else`, directly in the pass's body, comparing the index with `0` so that it holds on every pass but the first, or with `length - 1` so that it holds on every pass but the last, either way round. It runs one statement that appends to a variable with `+=` or calls a method of an object, reading nothing the pass declares.
- A piece is a later statement (or an earlier one, for every pass but the last) that appends to the same variable or calls a method of the same object, written alike.
- The rule names the line of a `continue`, or a `return` from the callback, before the piece, which ends a pass without it.
- A cut is `slice`, `substring` or `substr` of a variable, by a positive integer literal from the start, or from the end as `(0, -N)` or `(0, name.length - N)`, after a loop whose last write to the variable is a `+=` whose literal text at that end is at least `N` characters long. The rule says when the variable starts with text the cut would remove when no pass runs.

## Reporting and limits

Reports `positionalSeparator` on the guard and `trimmedSeparator` on the cut. No fixes.

This is a syntax rule. It compares sinks by their text, reads method names (`entries`, `forEach`, `slice`) without knowing the types they belong to, and reads `+=` without knowing it builds a string. Flags tracking the first pass, separators placed in a conditional expression, and arrays with a separator popped after the loop are not read.
