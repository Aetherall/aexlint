# no-repeated-conversion

Report a conversion written identically in several places.

**Status:** experimental. Built on the [decision model](../design/decision-model.md). Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-repeated-conversion": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Reported code

The same conversion of a variant value, written in two files:

```ts
// toolbar.tsx
item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };
// dropZone.tsx
dragged.target === "cell" ? { type: "cell", template: dragged.template } : { type: "row" };
```

When the conversion changes, every copy must be found and changed the same way.

## What counts

- A dispatch is a switch, if/else chain, or conditional chain testing a finite literal property, as in [no-external-variant-behavior](no-external-variant-behavior.md#what-counts).
- Its **constructions** are the object and array literals a branch builds from other fields of the tested value, with their destination: returned, assigned, stored in a variable, or passed as a given argument of a given callee.
- Two dispatches are copies when they test the same property, meaning the property has the same name and at least one shared declaration, have complete non-overlapping variant conditions, build constructions for the same set of values, and build identical construction forms for each value. An additional condition such as `enabled` or a test on another property makes the variant claim incomplete, so the dispatch is not compared.
- Constructions are compared by their [form](../design/decision-model.md#value) relative to the tested value: its name does not matter (`item.template` and `dragged.template` are the same read, also inside a template string or a callback, where another variable of the same name is not the tested value), literals are compared by value, and optional reads and calls remain distinct from ordinary ones. Static unique object keys can be reordered only with supported, non-accessor data reads or literal and boolean expressions as values. Spreads, computed keys, duplicate keys, accessors, and calls keep their order. External references and callees must resolve to the same symbol; names declared inside an expression must retain their spelling.
- Copies can be in the same file or in other project files of the TypeScript program, including unlinted files. A dispatch sharing only some of its branches with another is not a copy.

## Accepted code

A conversion written once, two conversions that differ in any key or value, conversions of different types that happen to look alike, and branches that do not build objects from the tested value. Near copies that drifted apart, such as the same action built with one field taken from a different place, are not reported.

Changing only an imported type can change the result: widening the tested property to `string` removes the finite vocabulary, so nothing is reported.

## Reporting scope

Reports `repeatedConversion` on the tested property of each dispatch that has at least one copy. The message names the property as written. The help lists the other copies, nearest first, and recommends writing the conversion once, with the owner of the type or as a shared function.

## Boundaries

Versioned serializers repeat conversions on purpose. Turn the rule off for them with Oxlint overrides, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries).

## Fixes

None. Choosing where the single conversion lives is a design decision.

## Limitations

- Copies are found through object literals with the same keys and constant values. Dispatches whose constructions are only arrays are not compared across places.
- Exact equality only: any renamed local variable, reordered call argument, or extra field makes two copies different.
- Types declared twice, for example a copied database document type, are different subjects, so conversions of them are not compared.
- Other tsconfig programs are not searched, so copies between, for example, a web and a native app with separate projects are not found.
