# no-default-outside-owner

Reports code that decides, outside a type's owner, what an absent optional property stands for. The decision is either a `??` or a destructuring default with a constant of the property's own type. The type doesn't say what absence means, so each reader decides for itself. The rule reports when two readers contradict each other, or when more than `max` readers outside the owner repeat the same decision. Copies drift, and a contradiction means the property already means two things.

```json
{ "rules": { "aexlint/no-default-outside-owner": ["warn", { "max": 1 }] } }
```

`max` is an optional non-negative integer, default 1, for repeated defaults outside the owner. With `0`, every default outside the owner is reported, even the only one.

## Example

Reported on both defaults:

```ts
// types.ts
export interface Shelf {
  readonly side?: "left" | "right";
}
// readers.ts
const side = shelf.side ?? "left";
const { side: mirrored = "right" } = shelf;
```

Reported on both reads with the default `max`, accepted with `{ "max": 2 }`:

```ts
const turns = knob.turns ?? 9;
const spins = (knob.turns ?? 9) + 1;
```

Accepted when one reader decides, or when the defaults are inside the owner: in the class declaring the property, or in the module declaring the interface or type alias. A getter such as `get placed() { return this.slot ?? "front"; }` is the owner resolving it once. A default inside the owner still counts as one side of a contradiction.

Only a dependency's type can make the difference. With `flow?: "low" | "high"`, `vent.flow ?? "low"` and `vent.flow ?? "high"` contradict each other. With `flow: "low" | "high"`, which is required, neither is a default.

## What counts

- A default is a `??` or `??=` whose left side reads a property, or a default in an object binding pattern, including a parameter's. Its value is a constant (a literal, an enum member or another single-value type) assignable to the property's type without `undefined` and `null`, so `?? null` and `?? 0` for a string property are fallbacks, not defaults.
- The property is optional, or typed with `undefined`. It is declared by a named class, interface or type alias the project establishes as an owner (see [no-fact-outside-owner](../no-fact-outside-owner/README.md)), and every declaration of it belongs to that owner. Properties of anonymous types, of `any`, and of unresolved types are skipped.
- Defaults are matched by the property's declaration, and by its type where it is read, across the program's project files: for `Field<T> { value?: T }`, defaults of `Field<string>` and `Field<boolean>` are apart. They are compared by their constant's type.

## Reporting and limits

Reports `conflictingDefault` on a default outside the owner that another default of the same property contradicts, naming up to three others. Otherwise it reports `repeatedDefault` when more than `max` defaults outside the owner give the same value. No fixes.

Fallbacks computed from other data and `||` defaults are not read. A fallback that only displays absence, such as `"-"` in an export, counts like any other default. So does the default of an options type that each function taking it chooses for itself, as `timeoutMs ?? 30_000` in one command and `timeoutMs ?? 60_000` in another: the types do not tell an operation's own default from what a value's absence means.
