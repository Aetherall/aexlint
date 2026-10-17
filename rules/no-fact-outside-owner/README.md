# no-fact-outside-owner

Reports facts about a type's values established outside the type's owner. A fact is what a test establishes about a property, such as "the order is cancelled". When its meaning changes, copies elsewhere keep the old one and nothing points to them.

- **Bypassed:** the owner has a predicate for the fact, and a test elsewhere rewrites it instead of calling it.
- **Unanswered:** the owner has no predicate for it, and more than `max` files outside the owner write the same test.

```json
{ "rules": { "aexlint/no-fact-outside-owner": ["warn", { "max": 4 }] } }
```

`max` is an optional positive integer, default 4, for unanswered facts.

## Example

Bypassed, reported on the test in `refund.ts`:

```ts
// order.ts
class Order {
  cancelled?: Cancellation;
  isCancelled() {
    return Boolean(this.cancelled);
  }
}
// refund.ts
if (!order || order.cancelled) throw new NotFound();
```

Unanswered, reported on `postedAt` in `leaflet.ts` when five other files test it:

```ts
export interface Leaflet {
  postedAt?: Date;
}
<Badge posted={!!leaflet.postedAt} />;
```

## What counts

- A test is a presence test or a literal comparison (`r.kind === "a"`, `||` of values, `includes` on a literal list) on a property, including through a variable that only names it (`const { kind } = r`).
- A predicate is a parameterless method, getter or arrow property whose whole body tests a property of `this`. Every declaration a call can run, including subclass overrides, must ask the same question.
- A helper predicate stands in for the owner: an exported function, static method or object member with one parameter whose whole body tests a property of it, optionally after a presence check.
- A test rewrites a predicate when it tests the same property declaration and splits its values the same way, in either direction. For presence, both remove the same missing values.
- The owner of an interface or type alias property is its declaring file; of a class property, the class and its subclasses. Unanswered facts count every project file of the program, including tests, and merge equivalent forms (`!!m.postedAt`, `m.postedAt !== undefined`).

## Accepted

- Tests inside the owner, on `this`, in a helper predicate's own file, or in the property's declaring file.
- A presence test whose value is then read narrowed in the same function: a boolean method cannot narrow. Likewise a value test on a union when the predicate is not a type predicate.
- For unanswered facts: `boolean` properties, anonymous object types, declaration and dependency types, and facts in `max` files or fewer.

## Reporting and limits

`bypassedPredicate` is reported on the test; help shows the call to write, with `!`, `?.` or `?? false` when needed to give the same result for a missing value. `unansweredFact` is reported once per property, on its name; the message lists each fact with its file count and help lists the files. No fixes.

`switch` cases are not tests. Renamed or default re-exports of helpers and other tsconfig programs are not found. A test on a variable destructured from a parameter has no object to call the predicate on and is not reported. Serializers and generated code may need [overrides](../no-repeated-value-group/README.md#reporting-and-limits).
