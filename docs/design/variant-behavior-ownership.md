# Variant behavior ownership

**Status: design specification. Declaration-based class, runtime-value, and module ownership, and conversions owned by either their source or destination, are implemented experimentally as [no-external-variant-behavior](../typed-rules/no-external-variant-behavior.md). Adapters and multi-file ownership remain unresolved.**

## Problem

Code outside a type's owner can reconstruct that type's behavior by inspecting its variant tag, unpacking variant-specific fields, and choosing how to construct, interpret, or transform the value.

The problem is not the presence of a switch, the number of possible values, or nested control flow alone. It is that knowledge of the variants lives outside the abstraction responsible for them. A caller must understand the representation instead of asking the type to perform an operation.

Adding a variant or changing its representation then requires finding consumers that implement this knowledge themselves. Even a single consumer can hold behavior in the wrong place; repetition compounds the problem.

## Agreed principle

**Behavior that understands a type's variants should belong to that type's owner. Callers select or invoke the operation, rather than reconstruct it from the tag and fields.**

A dedicated free function in a consumer module is not sufficient ownership. Moving each branch into an arbitrary helper is also insufficient. The desired change is to move the operation to the abstraction that owns the variants.

A static operation is one clear expression of ownership. An instance operation may also express it. The rule must establish ownership through declarations and symbols, not infer it from function names.

## Examples

### External transformation

Report a transformation implemented in a consumer module:

```ts
function transformFigure(figure: Figure, offset: Point): Figure {
  switch (figure.kind) {
    case "line":
      return Figure.from({
        kind: "line",
        from: translate(figure.from, offset),
        to: translate(figure.to, offset),
      });
    case "circle":
      return Figure.from({
        kind: "circle",
        center: translate(figure.center, offset),
        radius: figure.radius,
      });
  }
}
```

The named helper still requires its consumer module to know Figure's variants. Calling a constructor at the end does not transfer ownership of the transformation.

The intended caller is:

```ts
const transformed = Figure.transform(figure, offset);
```

The owner implements the operation once. The implementation can still dispatch internally; internal dispatch is not itself the violation.

### Embedded conversion

Report a variant conversion embedded inside a larger operation:

```ts
return new Notification({
  source:
    source.kind === "account"
      ? { type: "account", id: source.accountId }
      : { type: "document", id: source.documentId },
  text,
});
```

The larger operation should invoke an owned conversion entry point:

```ts
return new Notification({
  source: NotificationSource.from(source),
  text,
});
```

A conversion between two abstractions may be owned by its source or its destination; the implemented rule accepts either and reports a generic ownership message rather than choosing one. Explicit adapters still need definition. The example illustrates separation of the conversion from the enclosing operation, not a universal requirement to put every conversion on its destination.

### Owned construction

A single construction entry point owned by the abstraction can legitimately inspect its input variants:

```ts
class Recipient {
  static from(input: RecipientInput): Recipient {
    switch (input.kind) {
      case "identifier":
        return new IdentifierRecipient(input.identifier);
      case "field":
        return new FieldRecipient(input.fieldKey);
    }
  }
}
```

Consumers should reuse this entry point. Similar conversion logic repeated elsewhere remains suspicious even if the owned entry point exists.

### Selection without reconstruction

Do not treat a dispatch as a violation merely because it selects existing operations:

```ts
switch (value.kind) {
  case "text":
    return TextRenderer.render(value);
  case "image":
    return ImageRenderer.render(value);
}
```

A scalar comparison or constant lookup is likewise not sufficient evidence of misplaced variant behavior. Whether a particular dispatch duplicates routing already provided by a framework is a separate question.

## Candidate diagnostic

For an established ownership violation:

> This code implements behavior for `Figure` by inspecting its variants. Move the operation to Figure's owner.

The help should identify the inspected tag, fields used to implement the operation, and the known owner declaration. It should not merely demand helper extraction.

If the detector establishes only inline reconstruction, but cannot establish ownership, it must not present the stronger ownership claim as proven. Such sites remain research candidates until a narrower contract is validated.

Proposed span: the discriminant of a switch, or the first subject in a recognized conditional dispatch. Report once per dispatch, not once per arm.

No automatic fix: moving an operation can change imports, dependencies, visibility, generics, and evaluation behavior.

## Detection specification

### Candidate evidence

1. Recognize a switch, if-chain, sibling-if run, or conditional chain over a finite typed vocabulary.
2. Resolve the tested value and tag through the real type checker. Property names such as `type` or `kind` are not required.
3. Identify fields read from the tested value inside branches, distinguishing the tag from fields used to implement the operation.
4. Identify construction, interpretation, or transformation performed with those fields, rather than whole-value delegation to an existing operation.
5. Resolve the declarations that own the variants and the operation's enclosing declaration.
6. Report only when the operation is demonstrably outside an accepted owner boundary.

The product rule implements a narrower evidence chain: returned object/array construction using receiver fields, and ownership established through declarations. Broader interpretation, complete output data flow, and integration ownership still require review.

### Required distinctions

- Calling `Figure.from({...})` after unpacking and transforming Figure fields can still be external reconstruction.
- Calling `Figure.transform(figure, offset)` delegates the whole operation.
- A static method on an unrelated class is not automatically an owner.
- A file-local function is not automatically an owner, even when its name describes the operation.
- Dispatch within an established owner can be legitimate.
- Exhaustive dispatch can still implement behavior outside its owner.
- Sparse dispatch can be legitimate; vocabulary size does not prove misplaced ownership.
- Result construction alone does not prove variant behavior. An unrelated options object or event payload inside a branch is insufficient.
- Nested control flow alone does not establish an ownership violation.

## Boundaries and unresolved decisions

### Ownership

Classes are preferred when suitable, but module ownership is accepted for types without a class or explicit runtime owner. The implemented precedence is:

1. A class instance belongs to its class. Operations beside that class in the same module are not class-owned.
2. A type merged with a runtime value belongs to that runtime abstraction.
3. A named type-only alias or interface belongs to its declaring module or namespace. The module can expose operations and use internal helpers to implement them.

Imports and re-exports preserve the original owner. Simple reference aliases do not create a new owner. Namespace boundaries follow symbol identity, including reopened namespaces, rather than treating every declaration in a file as one abstraction.

The following still require an explicit contract:

- Choosing a shared runtime owner for unions whose members belong to different abstractions. A named union declaration supplies a module boundary, not proof that one constituent owns the entire operation.
- Dedicated operation modules within the same abstraction.
- Adapters owning conversions between independently owned source and destination types. Source and destination ownership are accepted.

Do not approximate these boundaries with filenames, class names, or matching text.

### Libraries and adapters

External types cannot always be modified. A local adapter or per-variant handler table can be an appropriate owner of the application's integration behavior.

Library provenance alone is not an exemption: core application logic can also operate on imported variants. Being declared in a dependency does not make every consumer an adapter.

Serializers and other approved integration boundaries can be excluded through explicit Oxlint overrides, consistent with existing typed-rule practice. Automatic recognition of adapters is not specified. Not every conversion is automatically exempt.

### Repetition

A construction or conversion entry point may be tolerated when implemented once. What constitutes the same conversion across sites still needs definition; identical tags or vocabularies do not establish identical behavior.

Cross-file duplication detection is a separate evidence requirement. A file-local check cannot claim that an operation exists only once.

### Existing rules

`no-procedural-value-dispatch` detects control flow inside branches. It does not establish ownership, and branch extraction can silence it without moving the behavior to its owner.

`no-repeated-value-group` and `no-repeated-value-policy` identify repeated decisions. They do not prove that a conversion belongs to the tested type.

The number of unnamed vocabulary values does not establish ownership. Do not use a sparse-dispatch threshold as a substitute for the ownership principle.

## Semantic limitations

Reconstruction syntax alone does not establish misplaced ownership: constructors and conversions can legitimately belong to the source or destination abstraction. Fields read during construction need not belong exclusively to one variant. Vocabulary size does not resolve these questions.

The product rule resolves class, merged type/value, and module or namespace owners through declarations. It supports inferred aliases, properties, and constrained generics where declaration provenance remains available. Missing ownership evidence remains a skip, not acceptance. Declaration-file and dependency owners remain skipped without deciding whether their consumers are legitimate adapters.

The detector does not follow receiver aliases or intermediate locals, establish complete output data flow, or prove that call arguments become returned output. Object construction inside a returned or assigned call can count even when the call discards that object. Supported default alternatives are examined only when the tested conditions are solely vocabulary tests; fallthrough switch statements are not examined.

Primitive-subject reconstruction, framework routing, dedicated operation modules, and explicit adapter ownership remain outside the established contract. Successful diagnostic mechanics do not establish accuracy, readability gains, or production readiness.

## Validation for implementation and expansion

Use real compiler programs and neutral fixtures to cover:

- The same transformation outside and inside its owner.
- Embedded conversion versus a reused owned entry point.
- Whole-value delegation versus constructor arguments built by the caller.
- Owner and unrelated-class static methods.
- Exhaustive and partial dispatches with the same operation.
- Imports, aliases, re-exports, generic variants, shadowing, and unresolved types.
- Module-owned unions, class/runtime ownership taking precedence over module ownership, and explicit adapter boundaries once their contract is settled.
- Unrelated object construction, scalar comparisons, and existing renderer selection.
- Representation-equivalent syntax changes that must not evade the diagnostic.

Then review fresh findings and legitimate counterexamples with the user. Keep diagnostic mechanics, behavioral correctness, and readability judgments separate. Do not expand or promote the narrow experimental rule to the full contract until each additional ownership boundary can be detected and its findings have been reviewed.
