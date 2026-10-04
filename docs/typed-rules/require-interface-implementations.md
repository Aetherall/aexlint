# require-interface-implementations

## Intent

Require every interface to be implemented by at least two classes. An interface separates a contract from the code that fulfils it. With a single implementing class, nothing can be substituted for that class, yet the separation still costs: the interface and the class change together, and readers following the interface's name reach the contract instead of the code. An interface no class implements is not a contract for classes at all.

This makes `interface` mean "a contract shared by several classes". Shapes of data, such as props, DTOs, configuration, and object literals typed by a function, are not class contracts. The rule reports them when they are declared with `interface`; a `type` alias of the same shape is not checked.

Status: experimental.

Oxlint 1.83.0 has no rule that counts the classes implementing an interface. `typescript/consistent-type-definitions` chooses between `interface` and `type` for every object type, without regard to implementations.

## Runtime and options

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/require-interface-implementations": "warn"
  }
}
```

No options. The required number of implementing classes is 2. Options other than an empty object produce a backend configuration error.

## Measurement

**Interfaces.** Each `interface` declaration in the linted file, including declarations nested in namespaces, functions, and `declare global` blocks. Declarations of one interface merged across several `interface` blocks are one interface. Not checked:

- interfaces that also have a declaration in a declaration file (`.d.ts`) or an external library, such as augmentations of `Window`, `Array`, or a package's types: the project does not own those contracts;
- interfaces merged with a class of the same name, which add members to that class rather than declare a contract.

**Implementing classes.** A class declaration or class expression implements an interface when:

- its `implements` clause names the interface, directly or through an import alias, re-export, or namespace (`implements ns.Repo`), with any type arguments (`implements Repo<User>`);
- its `implements` clause names an interface that extends the interface, at any depth;
- its `extends` clause names a class that implements the interface, at any depth.

Classes are counted by declaration, so one class reached through several of these paths counts once. Structural compatibility alone does not count: a class with the right members but no `implements` clause, an object literal typed as the interface, and a function returning one are not implementations. An `extends` clause that is not a class name, such as a mixin call, contributes nothing.

**Scope.** Classes are searched across the whole TypeScript program containing the linted file, including files that are not being linted. Declaration files and external libraries are not searched. Test files that belong to the program count like any other file, so a test double declared with `implements` is a second implementation.

## Reported code

```ts
// repository.ts
export interface Repository {
  find(id: string): Promise<User>;
}

// postgres-repository.ts
export class PostgresRepository implements Repository {
  /* ... */
}
```

`singleImplementation` is reported on the interface's name (`Repository`) in each of its declarations in the linted file:

```text
`Repository` is implemented only by `PostgresRepository` (postgres-repository.ts). Interfaces must be implemented by at least 2 classes.
help: With one implementing class, nothing can be substituted for it: the interface and the class change together, and readers following `Repository` reach the contract instead of the code.
```

An interface no class implements reports `unimplementedInterface` on its name:

```ts
interface ButtonProps {
  label: string;
}
```

```text
No class implements `ButtonProps`. Interfaces must be implemented by at least 2 classes.
help: Only `implements` clauses count, including those of base classes and of interfaces that extend this one. This rule reserves `interface` for class contracts. Redeclaring the shape with `type` satisfies that convention without changing how its values are constructed, validated, or changed.
```

Paths are relative to Oxlint's working directory. An anonymous class expression is named `(anonymous class)`.

## Accepted code

```ts
export interface Clock {
  now(): number;
}
export class SystemClock implements Clock {
  now() {
    return Date.now();
  }
}
// in a test file of the same program
export class FixedClock implements Clock {
  now() {
    return 0;
  }
}
```

Also accepted:

- `interface Animal`, `interface Pet extends Animal`, and two classes implementing `Pet`: each class implements both interfaces;
- `class Base implements Shape` and `class Square extends Base`: two classes implement `Shape`;
- `type Props = { label: string }`, which is not an interface;
- `declare global { interface Window { analytics: Analytics } }`, an augmentation of a library interface.

Cross-file example: the linted file declares `interface Shape` and `class Circle implements Shape`. Another file declares `interface Solid` and `class Cube implements Solid`. With `interface Solid extends Shape {}`, `Shape` has two implementing classes and is accepted. Changing only that file to `interface Solid {}` makes the linted file report `singleImplementation`.

## Fixes

Diagnostic-only. No fixes or suggestions are emitted. Whether to inline the interface into its class, turn it into a `type`, or add the missing implementation is a design decision the rule cannot make.

## Limitations

- **Data shapes are reported.** Under this rule's definition, every interface that describes data rather than a class contract is reported. Projects that use `interface` for shapes should enable it only where interfaces are class contracts, through `overrides`.
- **`unimplementedInterface` is a declaration convention.** It is satisfied by redeclaring the same shape with `type`, which changes no behavior. Neither diagnostic establishes whether the data has an owner that maintains its invariants, and neither declaration form is evidence either way. Handwritten copies of an upstream library's types are reported here like any other shape, but converting them to `type` keeps the copy; this rule does not detect that the copy can diverge from the authoritative declaration.
- **Implementations outside the program are not seen.** A library's public interfaces that consumers implement, interfaces implemented in another tsconfig project, and test doubles in files excluded from the tsconfig are not counted. A deliberate boundary interface, such as a port whose only production adapter is in another layer, is reported when its test doubles are not declared with `implements` in the same program.
- **Only nominal declarations count.** Object-literal and factory implementations, which are common in TypeScript, are not counted.
- **`implements` through a type alias or a class name** (`implements SomeAlias`, `implements SomeClass`) is not followed to the interfaces behind it.
- **One report per declaration.** An interface merged across several declarations in the linted file is reported on each.
- **Cost.** The first linted file in a program builds a syntactic index of every class with a heritage clause, shared by all invocations ([program-wide indexes](../native.md#program-wide-indexes)). An invocation whose file declares no checked interface returns immediately. Otherwise it resolves, with its own checker, the heritage clauses of every indexed class, so cost grows with the number of files declaring interfaces times the number of classes with heritage clauses.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`. Editor integration is not supported yet.
