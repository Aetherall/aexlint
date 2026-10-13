# no-fact-outside-owner

Report facts about a type's values that code establishes outside the type's owner. A **fact** is what a test establishes about a property, such as "the order is cancelled". Two cases are reported:

- **Bypassed:** the owner already has a predicate for the fact, and a test elsewhere rewrites it instead of calling it.
- **Unanswered:** the owner has no predicate for it, and code in many files outside the owner writes the same test.

**Status:** experimental. Built on the [decision model](../design/decision-model.md#predicates). Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-fact-outside-owner": ["warn", { "max": 4 }] }
}
```

**Options:** optional `max`, a positive safe integer; default **4**. An unanswered fact is reported when more than `max` files outside the owner establish it. Unknown keys and invalid values are configuration errors. The default is not a validated design boundary. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Reported code

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

`Order` already says what "cancelled" means. The caller writes its own version. When the owner's definition changes, for example to also treat expired orders as cancelled, this copy keeps the old meaning and nothing points to it.

```ts
// message.ts
export interface Leaflet {
  postedAt?: Date;
}
// in five or more other files
<Badge posted={!!leaflet.postedAt} />;
```

"Posted means `postedAt` is set" is decided in each of those files. `Leaflet` owns nothing about it, so a change to that meaning, such as unposting by clearing to `null`, must find every copy. The report is on `postedAt` in `leaflet.ts`, where an answer would live. Once `Leaflet`, or a function in its module, has a predicate for it, the bypassed case reports each file that still writes the test.

## What counts

- A **test** is a [presence test](../design/decision-model.md#presence) of a property read in a condition, `!` operand, or `&&`/`||` operand, `Boolean(r.p)`, or a comparison of a property with literal values that the model's classifier reads as one test, such as `r.kind === "a"`, `r.kind === "a" || r.kind === "b"` or `["a", "b"].includes(r.kind)`.
- A **predicate** is a method, getter, or arrow-function property without parameters whose whole body is a test of a property of `this`, declared in project code. A bare `return this.value` is a read, not a test. Every declaration a call can run must be such a predicate asking the same question in the same direction: the versions of union members and overrides in project subclasses count, so a predicate that a subclass redefines, such as an `isEmpty()` that also accepts `""` for text values, is not the same fact on a receiver that can be that subclass. Setters, overload signatures and abstract members do not run.
- A test on `r.p` rewrites a predicate when `r`'s type has that predicate, the predicate tests the same property declaration, and both establish the same fact: for presence, both remove the same missing values that the property's type can hold; for values, both name the same literal values and split the property's values the same way, in either direction. `r.kind !== "a"` rewrites `isA()`, but `r.shape === "circle"` does not rewrite a predicate testing `"square"` and `"triangle"`, even when those are the only other values: adding a value would make them disagree. When a predicate and its opposite both exist, the one accepting the same values is suggested.
- A **helper predicate** stands in for the owner when the type cannot or should not carry the predicate itself, such as plain data types or trait-like modules. It is an exported function, an exported arrow-function constant, a static method of an exported class, or a function member of an exported object constant, declared in project code, with one parameter, whose whole body establishes a fact about a property of that parameter. A leading `value !== undefined &&` or `value &&` is allowed: it only accepts a missing value. A test on `r.p` rewrites it when `r`'s type is assignable to the parameter's type and both establish the same fact. The type's own predicates are preferred.
- An **unanswered fact** is a fact about a property declared in a project interface, class, or object type named by a type alias, that no predicate or helper predicate of the type establishes, and that tests in more than `max` files outside the owner establish. The owner of an interface or type alias property is its module: the file declaring it. The owner of a class property is the class and its subclasses. Tests establishing the same fact are counted together, whatever their form: `!!m.postedAt` and `m.postedAt !== undefined` are the same fact when `postedAt` cannot be falsy otherwise. Every project file of the program counts, including tests. A comparison with literal values counts for a property declaration when it names a value the declaration's type can hold: on a union of item types, `item.kind === "stock"` is a fact about the member whose `kind` is `"stock"`, not about the others. Such a test silently goes wrong when a new member should also count as a stock item, and nothing points to it, so it counts like any other fact. [max-interpretation-spread](max-interpretation-spread.md) reports the same tests per file, by vocabulary.
- Comparisons of the same property joined by `||`, `&&` or `!` are one test, as the model's classifier reads them. `a.type === "image" || a.type === "video"` asks about both values, so it rewrites only a predicate testing both, not `isImage`. Tests of other values in the same chain stay separate: in `!seat || seat.kind === "taken"`, the comparison is a test of its own.

## Accepted code

- Tests inside a class that has the predicate, including its subclasses: the owner may read its own fields.
- Tests on `this`.
- A presence test followed by a read of the same expression in the same function whose type no longer includes the missing values, such as `if (order.cancelled === undefined) return; use(order.cancelled)`. That read depends on narrowing, and a boolean method cannot narrow. A later read that still includes them, such as `order.cancelled?.at`, does not keep the test.
- A value test on a union receiver when the predicate is not a type predicate (`this is …`), for the same reason.
- A truthiness test where the predicate compares with `undefined` and the property can hold `""` or `0`: they are different facts.
- Tests in a helper predicate's own file, or in a file declaring the tested property: the owner and its stand-in may read the fields.
- Functions that are not exported, or take more than one parameter, are not helper predicates.
- For unanswered facts:
  - Properties whose type, apart from `null` and `undefined`, is `boolean`: the property is its own answer, and testing it is reading it.
  - Properties of anonymous object types that are not the whole type of a type alias, and of types declared in declaration files or dependencies.
  - Facts established in `max` files or fewer.
- Changing only an imported type can change the result: removing the method from the class removes the report, and so does narrowing a helper's parameter type so the tested value no longer fits it. Adding a predicate to a type moves its unanswered fact to bypassed reports in each file.

## Reporting scope

Reports `unansweredFact` once per property, on its name in the declaring file. The message names the owner and each unanswered fact with its number of files, most first: the missing values a presence test splits off, such as "checked for `undefined`" or "checked for `undefined` and other falsy values", or the literal values a comparison names. Facts with different written values stay apart, as for bypassed predicates: `kind === "text"` and `kind === "stock"` are two facts even when they are the only values. When the same tests ask about a property that several types in the file declare, such as `parcel.file` on a union of parcel types, the fact is reported once, on the first type, and the message names the others. The help lists the files, nearest first.

Reports `bypassedPredicate` on the test. The message names the tested expression and the predicate, and for a helper predicate its file. The help shows the call to write instead, with `!` when the test asks the opposite question, such as `!order.isCancelled()` for `!order.cancelled`. When the tested value can be missing, the suggested call must give the same result as the test for a missing value: `order?.cancelled` becomes `order?.isCancelled()`, but with a predicate `isOpen()` testing `this.cancelled === undefined`, `order?.cancelled !== undefined` is `false` for a missing `order` while `!order?.isOpen()` is `true`. Where the test's value is kept rather than used only as a boolean, as in `{ gone: order?.cancelled !== undefined }`, the suggested call must give the same boolean: `order?.isCancelled()` gives `undefined` for a missing `order`, so the help suggests `order?.isCancelled() ?? false`, and a helper must return a boolean. When the results differ, cannot be worked out from the code, or the helper's parameter does not accept a missing value, the help says to check the value first.

## Boundaries

Serializers that copy state field by field may test fields directly on purpose, and generated code cannot call hand-written predicates. Turn the rule off for them with Oxlint overrides, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries).

## Fixes

None. Calling the predicate can change behavior when the predicate is later redefined, which is the point, but deserves review. Where an unanswered fact's answer belongs, and what it should be called, is a design decision.

## Limitations

- `switch` cases are not tests.
- Helper predicates re-exported under another name, default exports, and helpers in another tsconfig program are not found.
- Unanswered facts count files, not tests. A file establishing the fact many times counts once, and copies in other tsconfig programs are not counted.
- Two helper modules defining the same fact are not reported: each is a stand-in owner.
- A later read narrowed by another test in the same function also keeps the test.
