# no-forgeable-validated-result

Report an operation that checks data, then returns it as a plain object type that other code can produce without the check. Consumers cannot tell whether a value passed validation.

**Status:** experimental research candidate; not ready for promotion. Earlier evaluation found only about 7% of reports useful. Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-forgeable-validated-result": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Example

Reported on `parseName`:

```ts
type Name = { value: string };
export function parseName(value: string): Name {
  if (!value.trim()) throw new Error("empty name");
  return { value };
}
```

Accepted alternative, when the value needs its own owner:

```ts
export class Name {
  constructor(readonly value: string) {
    if (!value.trim()) throw new Error("empty name");
  }
}
```

## Measurement

The rule checks functions, methods, and function expressions or arrows initializing variables or properties in the linted file. Constructors, accessors and callbacks passed directly as arguments are excluded.

A rejection is a `throw` under an input-reading `if` or `switch`, or a call to a project function that rejects passed input. Nested functions, classes and `try` blocks are skipped. Project calls are followed across files; library/declaration-file bodies are not. Recursive cycles add no rejection by themselves. Calls returning an owned, non-forgeable value do not propagate rejection: that value already has an owner.

Presence and type-narrowing tests are excluded: nullish checks, object-only truthiness, `typeof`, `instanceof`, `in`, type guards and assertion functions. String or number truthiness can be a content check. Mixed conditions may still check content.

**Data must overlap.** Input includes parameters and locals initialized from input, including destructuring and loop bindings. The analysis follows property paths, aliases, returned expressions and data stored into local properties. Checking `input.allowed` does not overlap returning only `input.id`; returning all of `input` does overlap.

Method calls and computed keys read an unknown part of their receiver, not every specific child property. Project calls carry arguments read by their returns; bodyless methods carry none, while library calls and constructions carry object arguments, not primitive lookup keys. Conditional expressions carry their branches, not the selecting condition. This is an approximation, not proof of what a check means.

**Result type:** unwrap promises and remove `null`/`undefined`. The remaining type must be a project-owned object with stored data, not just methods. Aliases and inline object types qualify; arrays, tuples, functions, primitives, type parameters, `any`, `unknown` and `never` do not. For unions/intersections, every member must qualify and be forgeable. Ownership requires declarations in project source files, not libraries or declaration files.

**Forgeable:** not a class instance and without private/protected members, `#private` names or unique-symbol keys. Every class is treated as an owner, even with public fields or a public constructor that bypasses a validating factory. This is a design convention, not TypeScript's structural assignability test.

## Reporting scope

Reports `forgeableValidatedResult` on the operation's name. Help identifies the first relevant rejection in source order, including the called function's location when applicable. Wrapped results name the wrapper type rather than its checked field.

## Limitations

- Authorization and other operation preconditions can report whenever their data also reaches the result. Data overlap does not distinguish them from validation.
- Parsers, serializers, I/O records and immediately consumed values can legitimately remain plain objects. The rule does not inspect consumers or how long values live.
- Reassignments, rest parameters, callback data flow and `this` state are not followed. Neither are guard-then-return rejections, expression-only conditions, `Result` failures or rejected promises without `throw`.
- Library validators are not followed. Delegating wrappers can report because their project callees reject input, even when the wrapper adds no validation.
- `readonly` does not prevent reports; mutable class instances are accepted. A report asks for design review, not automatic conversion to a class or brand.
