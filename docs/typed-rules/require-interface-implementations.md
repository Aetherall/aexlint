# require-interface-implementations

Reserve `interface` for contracts shared by at least two classes. A single-class interface adds a separate declaration to maintain; plain data shapes can use `type` instead. This is a declaration convention, not a check that data has a responsible owner.

**Status:** experimental. Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/require-interface-implementations": "warn" }
}
```

**Options:** none; the required count is always **2**. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error.

See [typed-plugin setup](../../README.md#type-aware-setup-and-limits) for project requirements. Run `tsc` separately; editor integration is unsupported.

## Example

Reported if `SystemClock` is the only implementing class in the program:

```ts
export interface Clock {
  now(): number;
}
export class SystemClock implements Clock {
  now() {
    return Date.now();
  }
}
```

Accepted when a second class also implements the contract:

```ts
export class FixedClock implements Clock {
  constructor(private readonly time: number) {}
  now() {
    return this.time;
  }
}
```

## What counts

- Class declarations and class expressions count through explicit `implements` clauses. Imports, re-exports, namespace references and generic arguments are resolved.
- Implementing a child interface also counts for its parent interfaces. Extending an implementing class counts as another implementation. Each class declaration counts once.
- The search covers the whole TypeScript program, including unlinted source files and tests, but not declaration files or external libraries.
- All interface declarations in the linted file are checked, including nested and global declarations. Merged interface declarations share one count.
- Interfaces with any declaration in a declaration file or external library are excluded, as are interfaces merged with a class. This permits library augmentations such as `Window`.

## Reporting scope

Zero classes produces `unimplementedInterface`; one produces `singleImplementation`. Each report spans the interface name, once per declaration in the linted file, including merged declarations. The single-implementation message names the class and its file.

`type` aliases are not checked. Changing an interface to a type alias satisfies this convention without changing construction, validation or ownership of its values.

## Limitations

- Props, DTOs and other data-only interfaces report by design. Use `overrides` if this convention belongs only in part of your project.
- Structural compatibility, object literals and factory-returned objects do not count. A class needs a recognized `implements` path, not just matching members.
- `implements` through a type alias or class name is not followed to underlying interfaces. A mixin call in `extends` contributes nothing.
- Implementations in other tsconfig programs, downstream consumers or excluded tests are invisible. Public library contracts and deliberate boundary interfaces can therefore report.
- Two declared implementations do not prove that the abstraction is useful. Likewise, replacing a copied library interface with `type` does not remove the copy.
- Each file declaring checked interfaces resolves the program's indexed class inheritance clauses; large programs can be expensive.
