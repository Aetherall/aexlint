# Missed diagnostics

Found while redesigning `placeAnnotations` in pi-harness (`plugins/pi-feed/src/layout.mts`). The only type-aware finding on it is `splitOutcome`, and its help points to the weaker fix. The redesign took several steps, and each one answers a flaw no rule reports. Each section lists the flaw, the line it should fire on, the refactor it leads to, and what the model lacks to see it.

## The function

```ts
const rows = new Map<number, Row>();
const indices = lines.flatMap((line, row) =>
  !isImageLine(line) && stripTerminalSequences(line).trim() ? [row] : [],
);
if (!indices.length) return output;
for (const contribution of annotations) {
  const { annotation } = contribution;
  const size = visibleWidth(annotation.text);
  if (/[\r\n]/.test(annotation.text) || size === 0 || size > width) continue;
  let index = annotation.anchor === "heading" ? indices[0]! : indices.at(-1)!;
  let row = rows.get(index) ?? new Row(output.lines[index]!);
  if (!row.fits(contribution, width)) {
    if ((annotation.overflow ?? "hide") === "hide") continue;
    index = output.lines.length;
    output.lines.push("");
    row = new Row("");
  }
  row.add(contribution);
  rows.set(index, row);
}
for (const [index, row] of rows) {
  output.place(row.render(index, width, theme, hovered), index);
}
```

## The design it should reach

Two kinds of row were sharing one variable, one map and one loop:

- **Anchor rows** sit on the heading or the last line and collect annotations.
- **Overflow rows** hold a single annotation and are complete when created.

Once they are separated, the rest follows:

- A row carries its own index.
- The annotation answers questions about itself.
- A value that exists only when there are lines to anchor to owns the packing.
- A generator yields each row from the branch that makes it, so no merge follows the decision.

```ts
export function placeAnnotations(...): AnnotatedLines {
  const output = new AnnotatedLines(lines);
  for (const row of Anchors.of(lines)?.rows(annotations, width) ?? []) {
    output.place(row.render(width, theme, hovered));
  }
  return output;
}

class Anchors {
  static of(lines: readonly string[]): Anchors | undefined;

  *rows(annotations: readonly AnnotationContribution[], width: number): Generator<Row> {
    const visible = annotations.filter((entry) => entry.displayableWithin(width));
    const overflow = yield* this.#packed(visible, width);
    yield* overflow.map((entry, offset) => Row.own(this.#lines.length + offset, entry));
  }

  *#packed(contributions, width): Generator<Row, AnnotationContribution[]> {
    const rows = new Map<number, Row>();
    const overflow: AnnotationContribution[] = [];
    for (const contribution of contributions) {
      const index = contribution.atHeading ? this.#heading : this.#end;
      const row = rows.get(index) ?? new Row(index, this.#lines[index]!);
      if (row.fits(contribution, width)) rows.set(index, row.with(contribution));
      else if (contribution.ownsRow) overflow.push(contribution);
    }
    yield* rows.values();
    return overflow;
  }
}
```

Only `#packed` keeps state between annotations, because whether one fits depends on what its row already holds.

## 1. One collection, two kinds of entry

**Fires on** `rows.set(index, row)`.

The key reaches it from two places:

- `rows.get(index)`, so the entry may be looked up again;
- `output.lines.length`, a new number each time that nothing ever looks up.

The second kind of entry is a list stored in a map, and the code after the decision is shared only because both kinds go through it.

**Diagnostic.** "`rows` stores some entries under keys it looks up, and others under keys it never looks up: these are two collections."

**Leads to** keeping anchor rows in a map and overflow annotations in a list.

**Model gap.**

- A `Mutation` write records the receiver but not the arguments' roles, so a map's key and value are invisible.
- Nothing follows where a key comes from, or whether the same collection is ever read with it.

**Diagnosis.** A first attempt reported this as a false merge, the overflow branch steering the shared `row.add` and `rows.set` by giving `index` and `row` a different kind of value. With provenance read where each arm reaches the merge, that difference disappears: the branch appends `""` to `output.lines` right after reading its length, so `index` names an existing line, and the other arm's get-or-create can also bring a new `Row`. What remains specific to this branch is fabrication. The model now reads it: the `""` is a placeholder, its position flows through the keys of `rows` to `output.place`, whose summary stores into `output.lines` at that index, and no read in between can see it. The key is read too: `rows` is looked up by the anchor indices, while the entries stored under keys from `output.lines.length` are only reached by the final loop. `no-split-result` reports the decision as `fabricatedOutcome`, naming both.

## 2. The key passed back into the value it indexes

**Fires on** `row.render(index, …)` inside `for (const [index, row] of rows)`. `output.place(…, index)` repeats it.

A value that needs its own key to do its work owns that key. The map holds it outside, so every consumer has to pass the two together.

**Diagnostic.** "`row` is always used with the key it is stored under: make the key part of the value."

**Leads to** `Row.index`, `RenderedRow` placing itself, and generators yielding rows rather than pairs.

**Model gap.** Nothing relates a destructured map entry's key to its value, or notices that a method of the value receives the key.

## 3. A non-null assertion backed by a distant check

**Fires on** `indices[0]!` and `indices.at(-1)!`.

They hold only because of `if (!indices.length) return output` further up. Nothing carries that guarantee: moving or dropping the check leaves the assertions in place and wrong.

**Diagnostic.** "This assertion holds only because of the length check at line N: make a value that can only exist when the list is not empty."

**Leads to** `Anchors.of(lines)`, which returns `undefined` when there is nothing to anchor to, so the heading and end indices are always defined.

**Model gap.** Endings already tell whether a guard returns early. Still missing is a fact tying an assertion to the guard that makes it true. `no-repeated-presence-check` reads presence checks, but across calls, not between a guard and an assertion.

## 4. A default applied where the field is read

**Fires on** `(annotation.overflow ?? "hide") === "hide"`.

The default for an optional field of `Annotation` is decided in layout code, and a second reader must repeat it or contradict it. The tests on `annotation.anchor` and on `annotation.text` are the same kind of fact, made about another type's value.

**Diagnostic.** "`overflow` is resolved to its default here, outside the type that declares it: expose the resolved value from its owner."

**Leads to** `contribution.ownsRow`, `contribution.atHeading` and `contribution.displayableWithin(width)`.

**Model gap.**

- `no-fact-outside-owner` and `max-interpretation-spread` count repetitions across files, so a fact made once is never reported.
- A default applied at a read site is a narrower, single-site signal: `??` over an optional property, then a comparison with one of its literal values.

## 5. A position allocated by growing the output

**Fires on** `index = output.lines.length` followed by `output.lines.push("")`.

The output's length serves as a counter while the output is being built, and the empty line exists only to reserve the position. This is a weaker finding on its own. Its cost is the fresh keys behind §1.

**Diagnostic.** "`output.lines` is grown to number rows: number them where they are placed."

**Leads to** overflow rows numbered `lines.length + offset` after packing, with no write to the output during it.

**Model gap.** Liveness and writes see the `push` and the read of `length`. What's missing is a fact that one value's size is read and then grown, inside a loop, as a source of keys.

## 6. A filter written as an early `continue`

**Fires on** `if (/[\r\n]/.test(annotation.text) || size === 0 || size > width) continue;`.

The guard depends only on the element and on values the loop never changes, so it is a filter step written inline. This is the weakest finding, and a matter of taste unless it combines with the others: it is what gave the loop a third job.

**Diagnostic.** "This loop skips elements by a test on the element alone: filter them before the loop."

**Leads to** `annotations.filter((entry) => entry.displayableWithin(width))`.

**Model gap.** Endings know a `continue` targets the loop. Nothing says that a condition reads only the loop variable and invariant values.

## Order

§1 and §3 are the most general, and the most precise to build on writes, endings and liveness. §1 also fixes `splitOutcome`'s help. §2 and §4 are cheap and single-site. §5 and §6 matter mainly as evidence for §1.
