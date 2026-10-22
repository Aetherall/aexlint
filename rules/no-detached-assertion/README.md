# no-detached-assertion

Reports a non-null assertion on a list element or map entry that holds only because of a size or key check elsewhere. The types don't connect the assertion to the check. If the check moves, changes or goes away, the assertion stays and is wrong, and the type keeps saying the entry may be missing.

```json
{ "rules": { "aexlint/no-detached-assertion": "warn" } }
```

No options.

## Example

Reported on `indices[0]!` and `indices.at(-1)!`, with `noUncheckedIndexedAccess`:

```ts
const indices = lines.flatMap((line, row) => (line.trim() ? [row] : []));
if (!indices.length) return output;
for (const note of notes) {
  const index = note.heading ? indices[0]! : indices.at(-1)!;
  // ...
}
```

Reported on `counts.get(key)!`:

```ts
if (!counts.has(key)) return 0;
return counts.get(key)! + 1;
```

Accepted when the check opens the branch holding the assertion, as in `if (counts.has(key) && counts.get(key)! > 2)`, and when the entry itself is checked, so its type narrows:

```ts
const count = counts.get(key);
if (count === undefined) return 0;
return count + 1;
```

Accepted when the element type includes `undefined`, as in `(string | undefined)[]`: then the check says nothing about the value. Only an imported type can make this difference. With `type Names = readonly string[]`, `if (!names.length) return ""; return names.at(0)!;` is reported. With `readonly (string | undefined)[]`, it is not.

## What counts

- The assertion is on an element read with a literal index, on `at` with a literal (negative from the end), on the element at `length` minus a literal, on a map's `get`, or on an object's entry read by a key that is not an integer literal. The read's type includes `undefined` or `null`, and the type of the container's entries does not. Entries typed `any`, `unknown` or `never`, unresolved types, and other lookups such as `find` are skipped. Collections are the global `Array`, `ReadonlyArray`, `Map`, `ReadonlyMap` and `WeakMap`, including subclasses, generics and unions whose members agree.
- The check is a guard earlier in an enclosing statement list whose branch leaves (`return`, `throw`, `break`, `continue`) when the check fails, or a call to an assertion function declared `asserts condition`, such as `assert(rows.length > 2)`. It can also be a call to a project function guaranteeing the check once it returns: a guard at the top of its body that only throws, or an assertion call there, when the function removes no entries. The assertion holds only because the code reaches it. A check that opens a branch holding the assertion is accepted: an `if` or `else`, a conditional expression, or the right side of `&&` or `||`, as in `rows.length > 1 ? rows.at(1)! : 0`, since the assertion visibly depends on it. It reads the container's size, as a truthiness test or compared with an integer literal (such as `length > 1`, `size === 0` or `length < 2`), or tests the same value as the key with `has`, `in`, `Object.hasOwn` or `hasOwnProperty`. Negations, `&&` and `||` combine checks.
- The check must guarantee the entry exists. `length > 1` guarantees `at(1)` and `at(-2)`, not `at(2)`.
- The check can sit outside a callback around the assertion, provided an array or set method calls that callback with its elements.
- Nothing between the check and the assertion removes entries, changes the container through a method with an unknown body, or rebinds it. Loops and callbacks around the assertion count in full.

## Reporting and limits

Reports `detachedAssertion` on the assertion, naming the check and its line. No fixes.

Keys are compared as values: the same variable, `this` or property path, not two expressions that happen to be equal. A `const` of the function, or a variable it never rebinds, holding a size or a check counts as what it holds, as in `const count = rows.length; if (count < 2) return;`. Other variables, and checks in functions that return when they fail, are not read.
