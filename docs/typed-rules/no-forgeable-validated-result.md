# no-forgeable-validated-result

## Intent

Report an operation that checks the value it returns, rejecting some inputs, when the value's type lets any other code produce it without that check. The check establishes something about the value, but the type does not record it:

- code that receives the value cannot tell a checked value from one built elsewhere, so it either trusts every producer or checks again;
- every other place that builds or changes such a value must remember to call the check, and nothing points to a place that does not.

The state and the operation that enforces its invariants then have no common owner. The burden does not depend on how a structural type is declared: an `interface`, a `type` alias of an object literal, an intersection, and an inline object type are equally forgeable, because an object literal or a type assertion produces each of them. A class owns its instances: other code produces one through its constructor, and producing one any other way takes a deliberate object literal or assertion. A type with private members or a `unique symbol` brand can only be produced by code that has access to them.

The diagnostic describes the separation. It does not prescribe a remedy; a class, a branded type produced only by the checking factory, or keeping the check where values are consumed are all possible designs.

Status: experimental research candidate. It is not part of self-lint and is not ready for promotion: see [Evaluation](#evaluation). See [Limitations](#limitations).

Oxlint 1.83.0 has no rule that relates a function's rejections to the type it returns. `require-interface-implementations` is unrelated: it measures class contracts and is satisfied by declaration syntax.

## Runtime and options

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/no-forgeable-validated-result": "warn"
  }
}
```

No options. Options other than an empty object produce a backend configuration error.

## Measurement

**Checked operations.** Function declarations, methods, and function expressions or arrow functions that initialize a variable or a property, with a body, in the linted file. Functions passed directly as arguments, getters, setters, and constructors are not checked; a constructor's result is an instance of its class, which is owned.

**Input.** A parameter of the operation, including names bound by destructuring a parameter, and any local variable of the operation (not of a nested function) whose initializer reads input, including `for…of` and `for…in` bindings over input and destructured bindings. Variables are classified in source order from their declarations; later assignments are not followed. `this` is not input.

**Rejection.** The operation rejects input when, outside nested functions and classes and outside the `try` block of a `try` statement:

- a `throw` statement is inside an `if` whose condition reads input, or inside a `switch` whose discriminant reads input, between the `throw` and the operation's body, and the innermost `if` around it is not a narrowing test; or
- a call or `new` expression passes input as an argument, its own result is not unforgeable (below), and the resolved signature's declaration is a function, method, or constructor declared in a project source file that itself rejects input. This is followed across files and at any depth; recursion counts as not rejecting.

A narrowing test is a condition built only from `!`, `&&`, `||`, comparisons with `null` or `undefined`, truthiness tests of values whose type contains only object types, `null`, and `undefined`, `instanceof`, `in`, comparisons of `typeof`, and calls to type guards and assertion functions (whose signature declares `x is T`, `asserts x`, or `asserts x is T`), whether in a condition or called on their own. It establishes presence or a type, which the narrowed type already carries: `if (!found) throw` after a lookup and `if (model instanceof Draft) throw` before serializing are not rejections, while `if (!found || !found.model) throw` is. Truthiness tests of strings, numbers, and `unknown` are content checks.

A call whose result is unforgeable does not propagate the callee's rejection: the check belongs to that owner and is carried by its type, so using an owner (`new Session(id)` inside a lookup) does not make the caller a checking operation.

Functions and constructors from declaration files and external libraries, such as a schema library's `parse`, are not followed.

**Checked data and result data.** A rejection is reported only when the data it checks overlaps the data the result is built from. This is what separates a check of the returned value (a validation) from a check of whether the operation may run (a precondition): `if (template.isDefault) throw ...; return { id: template.id }` checks `template.isDefault` and returns `template.id`, so it is accepted.

Data is described by access paths: a root followed by property names (`command.actor.roles`). A root is a parameter, or a local whose value is new data rather than a rearrangement of other values: a local initialized by a call, a construction, or a literal that reads no other data. A local initialized by an access path (`const after = input as Payload`, `const { left } = input`) is an alias for that path; other locals hold the paths of their initializer. Only paths rooted at input count: a value the operation obtains without input, such as the current time, does not connect a check to the result.

- **What a condition checks.** The paths its expression reads, except narrowing tests among its `!`, `&&`, and `||` operands: `!site || site.teamId !== teamId` checks `site.teamId` and `teamId`, not `site` as a whole. A method call (`actor.roles.includes(role)`) or a computed key (`settings[key]`) reads an unknown part of its receiver (`actor.roles`).
- **What a call checks.** The paths the callee checks of its parameters, mapped onto the arguments: `ensureMember(command.actor)`, where `ensureMember` checks `actor.roles`, checks `command.actor.roles`. An argument that is not an access path is checked as an unknown part of its paths.
- **What the result is built from.** The paths read by the expressions of the operation's `return` statements, or by an arrow function's expression body.
- **Through conditionals.** A conditional expression is built from its branches; its condition only selects one.
- **Through calls.** A method call carries an unknown part of its receiver. A call to a project function carries an unknown part of the arguments its own `return` statements read; a project function or method without a body, such as an interface or abstract method, carries none. A call to a library function or a construction carries its object arguments; a primitive argument, such as an identifier used as a lookup key, carries none. A local initialized by a call keeps its own root, and what the call carried is attributed to that local as a whole, not to any of its properties.
- **Through mutation.** An assignment to a property of a local, or a method call on it (`out.values.push(value)`), stores the assigned or passed data into that part of the local. Reading that part or the whole local reads it; reading another property does not.
- **Overlap.** Two paths overlap when they have the same root and one chain is a prefix of the other. An unknown part of a path overlaps that path and its containing paths, and other unknown parts of it, but not a specific property below it: checking `actor.roles.includes(...)` does not overlap returning `actor.id`.

**Result.** The operation's return type, from its signature. A `Promise` (in an `async` function or a declared return type) is replaced by its awaited type, and `null` and `undefined` are removed. The result is checked when it is an object type that is not an array, a tuple, or a function type, or a union or intersection whose members all are. Type parameters, `any`, `unknown`, `never`, and primitives are not checked. A result must also hold state: an index signature, or a property that is not a method and whose type is not a function. A result made only of operations, such as a visitor object returned after validating options, has no state for the check to establish.

**Owned.** The result's type alias, otherwise its symbol, has every declaration in project source files. For an intersection or union without an alias, every member is owned. A type without an alias or symbol is not checked.

**Forgeable.** The result is not a class instance type, and no property of it is declared `private` or `protected`, has a `#private` name, or has a `unique symbol` key. For a union or intersection, every member is forgeable. TypeScript itself treats a class whose fields are all public as structural: an object literal with the same members is assignable to it. This rule nevertheless treats every class as the owner of its instances, because forging one takes a deliberate literal or assertion where a class is expected, while a structural type is ordinarily built from literals. A validating static factory next to a public constructor is therefore accepted, although `new` bypasses the factory's check. **Unforgeable** means object-typed, owned, and not forgeable.

## Reported code

```ts
export interface Payload {
  readonly model: string;
  readonly tier?: string;
}

export function applySettings(before: Payload, input: unknown): Payload {
  if (!input || typeof input !== "object") throw new Error("payload must be an object");
  const after = input as Payload;
  if (after.model !== before.model) throw new Error("cannot change model");
  return { ...after };
}
```

`forgeableValidatedResult` is reported on the operation's name (`applySettings`):

```text
`applySettings` rejects some inputs, then returns `Payload`, which other code can produce without that check.
help: Code that receives a value of type `Payload` cannot tell whether it passed `applySettings`, and each other place that builds or changes such a value must repeat the check or skip it. Rejection: throw at line 6.
```

The help names the first rejection in source order: a `throw` with its line, or a call with the called name, its file, and line. Paths are relative to Oxlint's working directory.

Also reported, with the same `Payload` check:

- `type Payload = { model: string }`, `type Payload = Record<string, unknown>`, `type Payload = Base & { model: string }`, and an inline return type `{ model: string }`;
- `async function load(...): Promise<Payload>` that throws on its input;
- a method `bind(contract, tool)` that calls `this.assertCompatible(contract, tool)`, which throws on its parameters, and returns `{ ...tool }`;
- a check of a value derived from the result's data: `const total = sum(input.items); if (total > 10) throw ...; return { items: input.items }`, where `sum` returns data read from its parameter;
- a check of each element stored into a returned local: `for (const value of input) { if (!value.trim()) throw ...; out.values.push(value); } return out`.

## Accepted code

```ts
export class Payload {
  readonly #model: string;
  constructor(input: unknown) {
    if (typeof input !== "string") throw new Error("model must be a string");
    this.#model = input;
  }
}

export function parsePayload(input: unknown): Payload {
  return new Payload(input);
}
```

Also accepted:

- any class instance result, including a class with public fields: `class Name { constructor(readonly value: string) {} static parse(input: string): Name { if (!input) throw ...; return new Name(input); } }`, and a validating constructor of such a class;
- a result type with a `unique symbol` key, such as `{ readonly [brand]: true; model: string }`;
- a throw that does not depend on input, such as `if (!initialized) throw ...` on module state, and a throw after a narrowing test only, such as `if (!found) throw ...` after a lookup or `if (model instanceof Draft) throw ...` before serializing;
- preconditions whose checked data the result does not contain: a role check on the caller, a duplicate check on loaded records, a state check on a loaded entity (`template.isDefault`) followed by returning another of its properties or the lookup key;
- a result chosen by the check but not built from the checked data, such as a constant;
- a rejection that happens only while constructing an unforgeable owner, such as `{ owner: new Owned(id) }` where `Owned` is a class;
- a result made only of methods or function-valued properties;
- a throw inside a callback, inside a `try` block, or only after a guard that returns early (`if (valid) return value; throw ...`);
- rejection only inside a declaration-file function, such as `schema.parse(input)`;
- a result type declared in a declaration file or external library;
- a result that is a primitive, an array, a function, or a type parameter;
- type guards and assertion functions, which return no object.

Cross-file example: the linted file declares `export function bind(tool: Tool): Tool { assertTool(tool); return { ...tool }; }` and imports `assertTool` from another file. When `assertTool(tool: Tool)` throws on `tool.name`, `bind` is reported. Changing only that file so `assertTool` logs instead of throwing makes `bind` accepted.

## Evaluation

Labeled evaluation, 2026-10-04, with the boundaries above. Every report from four codebases was read with its callers and labeled: an application backend of about 4,600 files, two smaller tools, and this repository. Corpus names and per-report labels are kept outside the repository.

| Label          | Reports | Patterns                                                                                                                                                                                                                                                                                                                                                                                |
| -------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Useful         | 4       | Validated values that are retained and passed on: three are the cases that motivated the rule, one is a field configuration whose preparation another code path is documented to skip                                                                                                                                                                                                   |
| Debatable      | 8       | Verified identities unpacked at once by one consumer, private helpers consumed by sibling methods, one-off parsers                                                                                                                                                                                                                                                                      |
| False positive | 42      | Serialization projections (13, 7 of them only through a nested serializer that throws), preconditions whose entity data reaches the response (12), I/O records (5), emptiness or registry misses as "not found" (4), parsers whose result is owned by its consumer (3), delegating authentication branches (3), a boundary projection after the owner was introduced (1), test code (1) |

Precision is about 7% counting only useful reports. The candidate does not meet a promotion bar. The patterns that dominate the false positives (rejection propagated through delegating and recursive calls, preconditions on loaded entities, checks the result type already encodes, values consumed immediately) are recorded under [Limitations](#limitations); narrowing them further is research, not a promotion step.

## Fixes

Diagnostic-only. Whether the value needs an owner, a brand, or no change is a design decision.

## Limitations

These boundaries are provisional:

- **Data overlap approximates what a check is about.** A rejection can establish two different things. A _validation_ establishes that the returned value is valid: the summary text is not empty, the settings change only allowed keys. A _precondition_ establishes that the operation may run at all: the caller has a role, no duplicate exists, a default template cannot be renamed. A precondition says nothing about the value returned afterwards, so the report's burden does not apply. Both look alike, so the rule uses the data relation in [Measurement](#measurement) instead, and that relation is not the distinction itself. Preconditions are still reported when the checked data reaches the result: an identifier checked for access and also returned, a checked entity returned whole or serialized, or a lookup through a project function whose body returns data built from its key. Validations are missed when the result is built from data the check does not read, such as a result chosen by the check, data obtained through an interface method, or a value passed through a library or project function whose return does not read it. Remaining preconditions can be suppressed per operation.
- **Parsers, decoders, and boundary projections.** A parser that rejects malformed input returns a plain record, and an owner that projects a plain object for an external API after checking it is reported for the projection. Whether such records need an owner depends on how long they are retained and where else they are built, which this rule does not measure.
- **I/O failures count too.** A wrapper that rejects a failed process or a malformed file returns a record whose only "invariant" is that the I/O succeeded; it is reported like a validation.
- **Not every rejection form or data flow.** Guard-then-return patterns, `Result`-style returns (`{ ok: false }`), rejected promises created without `throw`, assertion calls on `this` state, reassigned variables, conditions inside expressions (`cond ? x : fail()`), data captured by callbacks, `this` state, and rest parameters are not followed. Library validators are not followed, so schema-derived data built by `parse` and then returned as an owned forgeable type is not reported.
- **Mutability is not measured.** A forgeable result is reported whether or not its properties are `readonly`, and an unforgeable result whose state can still be changed is accepted.
- **Forgeability is per type, not per use.** A deliberately plain record, such as a serialization projection or options normalized once and consumed immediately, is reported when its producer rejects input.
- **Anonymous callbacks are not checked.** A validating arrow function passed directly to another call is not an operation under this rule.
- **Rejection propagates through delegation.** A method that only delegates to, or recurses into, a function that throws (nested serializers, authentication branches) is reported like the function that checks.
- **Use is not measured.** A checked value destructured at once by its only caller, or handed to a class constructor that owns it, is reported like a value retained and passed between operations, which is where the burden appears.
- **The report names the wrapper.** When the checked value is one field of a returned wrapper object, the report names the wrapper's type rather than the field's.
- **Cost.** Calls resolve their signatures, and calls to project functions analyze the called function's body, memoized per invocation, so a helper graph is analyzed once per linted file.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`. Editor integration is not supported yet.
