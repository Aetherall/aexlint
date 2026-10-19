# no-split-result

Reports an outcome split across several values that are set together, or never together, without anything stating it. Only the code that sets them knows which combinations go together; every reader restates or guesses that pairing, and a change to what one place sets breaks the others silently. The split shows up in two places: the optional fields of a function's result, and the locals a decision's branches set for the code after it.

```json
{ "rules": { "aexlint/no-split-result": "warn" } }
```

No options.

## Result fields

A function result whose optional fields its returns always set together, or never together. The result type allows combinations no return builds, so each reader tests the fields one by one and decides what a partial combination means.

### Example

Reported on `: Digest`: `summary` and `tokens` come together, and never with `warning`.

```ts
interface Digest {
  summary?: string;
  warning?: string;
  tokens?: number;
}
function digest(mode: Mode, text: string): Digest {
  if (mode === "off") return {};
  if (mode === "local") return { summary: text, tokens: text.length };
  return { warning: "remote digest unavailable" };
}
```

Reported on `: Link`: `label` and `url` are set together or not at all.

```ts
const link = (path: string | undefined, internal: boolean): Link => {
  if (!path) return {};
  if (internal) return { label: path, url: `/pipelines/${path}` };
  return { label: path, url: `https://ci.example/${path}` };
};
```

Accepted: `{ summary: string; tokens: number } | { warning: string }` as the result, an `undefined` result instead of `{}`, or returns that build several combinations of the same fields (`{ width }`, `{ height }`, `{ width, height }`).

Only the result type decides: with `type Reading = { celsius: number } | { fault: string }` imported from another file nothing is reported; changing that file to `{ celsius?: number; fault?: string }` reports the function.

### Fixing it

Declare the combinations the returns already build. The function usually compiles unchanged; only the type and its readers change.

```ts
interface Upload {
  state: "queued" | "sending" | "done" | "failed";
  sent?: number;
  error?: string;
}
function next(event: UploadEvent): Upload {
  switch (event.type) {
    case "queued":
      return { state: "queued" };
    case "progress":
      return { state: "sending", sent: event.sent };
    case "finished":
      return { state: "done", sent: event.sent };
    case "failed":
      return { state: "failed", error: event.message };
  }
}
function label(upload: Upload) {
  if (upload.state === "failed") return `Failed: ${upload.error ?? "unknown error"}`;
  return `Sent ${upload.sent ?? 0} bytes`;
}
```

becomes:

```ts
type Upload =
  | { state: "queued" }
  | { state: "sending" | "done"; sent: number }
  | { state: "failed"; error: string };
function label(upload: Upload) {
  if (upload.state === "failed") return `Failed: ${upload.error}`;
  if (upload.state === "queued") return "Queued";
  return `Sent ${upload.sent} bytes`;
}
```

`next` is unchanged. The fallbacks `?? "unknown error"` and `?? 0` were answers for states no code builds: with the optional shape they compile, and when a bug does produce such a state they show wrong data instead of failing. The union also made `label` handle a queued upload, which it showed as "Sent 0 bytes": `sent` does not exist there, so the old last line no longer compiles.

Other shapes:

- Fields set together or not at all: group them in one optional value, `{ link?: { label: string; url: string } }`, or return `undefined` instead of `{}`.
- The shape depends on which branch of a `switch (this.kind)` built it, and later methods switch on `this.kind` again with `!` on the fields: put the tag in the result (`{ position: "refining"; subject; prefix }`) and switch on that, or give each kind its own object.

"The optional shape is simpler" holds for the declaration only. The cost moves to every reader: each one tests the fields one by one, adds a fallback or a `!` for combinations that cannot occur, and must be found by hand when a state or a field is added, since the type says every combination is valid.

### What counts

- A function with a body whose result, awaited and without `null` and `undefined`, is one object type declared in the project: not a union, an intersection, a callable type, `any`, `unknown` or an unresolved type. A result with literal types per return, as TypeScript infers for several object literals, is a union and is not read.
- Every return gives an object literal, a conditional between them, `null`/`undefined`, or another value, such as a call's result, whose type is one object type assignable to the result. At least three objects are returned. With two, any fields that differ look paired by chance.
- A field is optional when it is marked `?` or its type includes `undefined`. For each optional field, a literal sets it, leaves it out (missing or `undefined`), or may set it: a value whose type includes `undefined`, `any` or `unknown`, or a spread whose type has the field. A field some literal may set is not read.
- Fields set by some literals and left out by the others are grouped by the literals that set them. A group of two or more fields is set together or not at all; two groups no literal sets together are never set together.

### Reporting and limits

Reports `splitResult` on the written result type, otherwise on the function, once per function, listing every pairing. No fixes: the union's members may need names.

A returned value other than a literal is read by its type: a field its type declares as required is set, an optional one may be set, and a missing one is left out. Values of a union type, `any` or `unknown` stop the rule. Correlations between fields of a type used by several functions are not read. Readers of the result are not inspected: the rule reports what the type allows, not a reader that mishandles it.

## Values a decision leaves

A decision whose branches set several values that the code after it reads. The decision has one outcome, but it is left as separate values: the code after it reads them apart and relies on a pairing only the branches state.

When a branch also makes up what the code after the decision expects, a [fabrication](../../docs/design/decision-model.md), the report is `fabricatedOutcome`: the branch produces a different kind of result from the others, and fabricates what the shared code needs so it can treat that result like theirs. It appends a placeholder that only the code after the decision replaces, or gives that code a key to store entries of a local map under that no lookup of the map uses, while lookups do reach its other entries.

### Example

Reported on `!shelf.fits(width)`: a full shelf sends the item to a new shelf at a new index, and the last two lines store whichever `index` and `shelf` survived.

```ts
let index = entry.failure ? 0 : labels.length - 1;
let shelf = shelves.get(index) ?? new Shelf(labels[index]);
if (!shelf.fits(width)) {
  if (!entry.overflows) continue;
  index = labels.length;
  labels.push("");
  shelf = new Shelf("");
}
shelf.items.push(entry.value);
shelves.set(index, shelf);
```

Reported as `fabricatedOutcome` on the width check: a note too wide for the last line gets a line of its own, so the branch appends `""` to `grid.lines` under the index it was about to have, and the last loop replaces it through that index. The shared `rows.set(index, row)` stores two kinds of row: rows looked up by the last line's index, and rows under new indices no lookup uses. The placeholder and the new index exist only so the new line can pass for an existing one.

```ts
for (const note of notes) {
  let index = lines.length - 1;
  let row = rows.get(index) ?? [];
  if (row.join(" ").length + note.length > width) {
    index = grid.lines.length;
    grid.lines.push("");
    row = [];
  }
  row.push(note);
  rows.set(index, row);
}
for (const [index, row] of rows) grid.place(row.join(" "), index);
```

Reported on `names[i] === wanted`: the flag and the position leave the loop together through its `break`.

```ts
let found = false;
let position = -1;
for (let i = 0; i < names.length; i++) {
  if (names[i] === wanted) {
    found = true;
    position = i;
    break;
  }
}
report(found, position);
```

Reported on `shape`: each case picks an area and a label.

```ts
let area = 0;
let label = "";
switch (shape) {
  case "circle":
    area = Math.PI * size * size;
    label = "round";
    break;
  case "square":
    area = size * size;
    label = "square";
    break;
}
report(area, label);
```

Accepted: each branch uses its own outcome, so nothing crosses the end of the `if`.

```ts
if (shared.fits(width)) {
  shared.items.push(entry);
} else if (entry.overflows) {
  own.push(Shelf.of(entry));
}
```

Also accepted: a branch that sets one value (`let label = "a"; if (formal) label = "b";`); branches that return or throw; values every way assigns again before reading them; values only read after an enclosing loop, such as accumulators, or set by a loop inside the branch; methods a branch calls on data, such as `registry.set(key, entry)`, which are commands rather than results; and a decision followed by a `return`, which builds the one value.

### What counts

- An `if` or `switch` statement in a statement list, including an `else if` chain, or a conditional expression run as a statement.
- Its left values, as the [decision model](../../docs/design/decision-model.md) defines them: data its branches rebind (by assignment, compound assignment, increment, decrement or destructuring) whose root is a variable or parameter of its function declared outside it, or `this`, when the statements after it read that data before every way assigns it again. Data a branch rebinds before a `break` counts when the statements after the loop, `switch` or labeled statement it leaves read it.
- How each branch can end comes from its [ending](../../docs/design/decision-model.md): a `return`, a `throw`, a call returning `never`, or a `break` or `continue` to another statement takes its writes elsewhere.
- Several fields of one root are one value: the object already holds them together. Two or more values report.

### Reporting and limits

Reports `splitOutcome`, or `fabricatedOutcome`, on the condition, or the expression a `switch` tests, once per decision, listing the values in the order the branches write them. No fixes.

Writes in callbacks are not read, since when they run is not known. A method called on data is never a write the branch decides, even one that changes it, such as `push`. A call leaves only when its type is `never`, so a branch ending in an unresolved call, or in one typed `void` that always throws, is read as reaching the end. Reads in callbacks after the decision count. The rule does not judge how tightly the values are coupled: a clamp setting a position and a remainder reports like a flag and the result it guards.
