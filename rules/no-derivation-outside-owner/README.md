# no-derivation-outside-owner

Reports a value computed from a property of a type, by the same function, in more than `max` functions outside the type's owner. Each reader derives the value itself, and they agree only as long as every copy computes it the same way: nothing ties the copies together, so when the derivation changes, the copies that are missed disagree with the rest. The value belongs to the type, as a member of its owner, of a type holding it, or a function beside it, so readers ask for it instead of computing it.

```json
{ "rules": { "aexlint/no-derivation-outside-owner": ["warn", { "max": 2 }] } }
```

`max` is an optional positive integer, default 2: how many functions may derive the value before each of them is reported.

## Example

Reported on each `slug(parcel.label)`: three functions derive the slug of a parcel's label, and a fourth that slugs it differently would break the links.

```ts
// types.ts
export interface Parcel {
  readonly label: string;
}
export function slugOf(text: string): string {
  return text.trim().toLowerCase().replaceAll(" ", "-");
}

// readers.ts
import { slugOf as slug, type Parcel } from "./types.js";
export const link = (parcel: Parcel) => `/parcels/${slug(parcel.label)}`;
export const heading = (parcel: Parcel) => `# ${slug(parcel.label)}`;
export const anchor = (parcel: Parcel) => "#" + slug(parcel.label);
```

Accepted: the same derivation in two functions, or in any number of members of the owner (`get slug() { return slugOf(this.code); }`).

Only a dependency's type can make the difference. With `declare function gauge(thread: string): number`, three readers of `gauge(spool.thread)` are reported. With `declare function gauge(thread: string): boolean`, they are not: a boolean is a test, which [no-fact-outside-owner](../no-fact-outside-owner/README.md) weighs instead.

## What counts

- A derivation is a call whose only argument reads a property, as `f(value.property)`, outside the property's owner (see [the glossary](../../docs/design/decision-model.md)). The property holds instance data: a property signature, or a property declaration or getter that is not static. Its type has no call signatures, and every declaration of it belongs to one owner, so a property read through a union of unrelated types is not one.
- The callee computes from its arguments alone: a function, an arrow function, or a static method or a method of an object literal, called directly or through an import alias. A method of an instance, such as `store.load(item.id)`, also reads the instance's data, so it is a lookup rather than a derivation.
- The call makes no effect and its result is used and is not a boolean, `void` or `never`. The body of a project function must make no effect on data declared outside it, such as module state; what the functions it calls do to such data is not read.
- Derivations are the same when they call the same declaration on the same property declaration, across the program; the property of a generic type is the same for every instantiation. A reader is the function a derivation is written in, or its file at the top level.
- `any`, `unknown`, `never` and unresolved types give no property declaration, so their reads are not derivations.

## Reporting and limits

Reports `repeatedDerivation` on each derivation of a value more than `max` functions derive, naming the property, its owner, the callee and up to three other readers. No fixes.

Only calls with one argument count: `format(item.amount, "EUR")` and method chains on the property (`item.label.trim()`) are not read. Keyed lookups through free functions, such as selectors given an id, read like derivations, since types do not tell a key from a value. Test helpers repeat derivations on purpose; exclude test files with overrides when they report.
