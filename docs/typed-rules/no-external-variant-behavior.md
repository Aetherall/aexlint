# no-external-variant-behavior

Report variant representations reconstructed outside a known type owner.

**Status:** experimental. Implements declaration-based class, runtime-value, and module ownership from the [variant behavior ownership design](../design/variant-behavior-ownership.md). Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-external-variant-behavior": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Reported code

A caller implements Figure's transformation by inspecting its tag and constructing results from its fields:

```ts
function transformFigure(figure: Figure) {
  switch (figure.kind) {
    case "line":
      return { value: figure.start + 1 };
    case "circle":
      return { value: figure.center + 1 };
  }
}
```

Moving this code to a named function in a consumer module does not establish ownership. An unrelated class's static method is not an owner either.

Prefer a class-owned operation when a class owns the type:

```ts
const transformed = Figure.transform(figure);
```

The implementation on Figure may still dispatch internally. A free function alongside the class is still external to the class: sharing its module does not replace class ownership.

## Ownership

Ownership follows declarations and checker symbols, not naming conventions or directory proximity:

1. A class instance belongs to its class. Class ownership takes precedence over module ownership, including through type aliases and generic constraints.
2. A type merged with a runtime value belongs to that runtime abstraction. Operations inside its initializer or merged namespace are owned; merely sharing its file is insufficient.
3. A named type alias or interface without a runtime owner belongs to its declaring module or namespace. The module's API and internal implementation can own operations on that type.

For example, a type-only union and its operation can share a module:

```ts
export type Figure = { kind: "line"; start: number } | { kind: "circle"; center: number };

export function transform(figure: Figure) {
  switch (figure.kind) {
    case "line":
      return { value: figure.start + 1 };
    case "circle":
      return { value: figure.center + 1 };
  }
}
```

Consumers call this operation instead of reconstructing its branches themselves. Importing or re-exporting Figure does not transfer ownership to a consumer or barrel module. Simple reference aliases follow the original declaration. Inferred aliases, typed properties, and constrained generics are resolved where the checker retains their declaration evidence.

Namespace boundaries use namespace symbols. Reopened declarations of the same namespace share ownership; an unrelated namespace in the same file does not. Global script declarations without a module or namespace are skipped.

### Conversions

A result has a destination type: an object or array literal's contextual type, such as a declared return type, variable annotation, assignment target, or parameter, or otherwise the type of the constructing expression. When the destination resolves to a project-local owner other than the source's, the dispatch is a conversion between two abstractions. Either owner may implement it:

```ts
export class Shape {
  static from(figure: Figure) {
    switch (figure.kind) {
      case "line":
        return new Shape({ size: figure.start });
      case "circle":
        return new Shape({ size: figure.center });
    }
  }
}
```

The same dispatch in a consumer module is reported as a conversion. A conversion is accepted inside any destination owner when its arms construct several. Anonymous, library, declaration-file, and unresolved destinations, and the source type itself, keep the source-owner diagnostic.

Classes remain the preferred design when suitable. Module support is a fallback ownership boundary, not a requirement to replace classes with free functions. This rule does not decide whether introducing a new class would be worthwhile.

## What counts

- A dispatch is a switch, conditional chain, if/else-if chain, or run of consecutive sibling ifs testing the same finite literal vocabulary. It needs at least two arms, at least one explicitly tested. A switch `default`, trailing else, or final ternary alternative counts as an arm for the remaining values when every tested condition is solely a vocabulary test; a condition combined with other operands, such as `figure.kind === "line" && ready`, leaves its alternative unexamined. Vocabularies follow [no-repeated-value-group](no-repeated-value-group.md#what-counts).
- The tested subject must be a property access. No special tag names are required.
- The receiver must have a known project-local owner as described above.
- At least one arm returns, assigns, or initializes a variable with an object-valued expression containing an object or array literal that reads the tested receiver's non-tag fields. Object literals inside constructor or call arguments of that expression count. Nested function and class bodies do not count toward construction evidence.
- Receiver identity combines normalized expression text and checker symbol identity. Shadowed identifiers with the same spelling are different receivers.

## Accepted and skipped code

Accepted selection delegates the whole value to existing operations:

```ts
switch (figure.kind) {
  case "line":
    return Figure.transform(figure);
  case "circle":
    return figure.transform();
}
```

Scalar comparisons, constant result objects, objects passed only to calls whose results are discarded, such as logging, and returned callbacks are not sufficient construction evidence.

Declaration-file, ambient, and dependency owners are skipped because an integration boundary is not established. A dependency is a file whose real path is inside `node_modules`; workspace packages linked into `node_modules` count as project code. Unresolved or erased declaration provenance is also skipped. These skips do not mean the code has been judged clean.

Changing only an imported type can change the result: widening a class's tag from a literal union to `string` removes the recognized dispatch. Removing only a merged runtime owner leaves a type-only union whose declaring module owns it instead. Neither change requires modifying the consumer.

## Reporting scope

Reports once per dispatch on its first tested subject. `externalVariantConversion` applies when results have project-local destination owners other than the source. Its message is generic: "This code does a semantic transformation that weakens ownership." with help "Find a suitable owner that fits the model or reconsider the approach." Otherwise the message is `externalVariantBehavior` naming the source owner. The help lists the non-tag fields used in result construction and recommends each owner as a class/runtime abstraction or declaring module, depending on its resolved boundary. Statements falling through from one case into the next are not examined.

## Boundaries

Use explicit Oxlint overrides for approved serializers or adapters, as shown for [no-repeated-value-group](no-repeated-value-group.md#boundaries). A library declaration is not proof that its consumer is an adapter; this implementation simply skips dependency ownership it cannot integrate.

## Fixes

None. Moving the operation to its owner can change dependencies, imports, visibility, and generic constraints. Calling a constructor after unpacking fields is not necessarily delegation of the entire operation.

## Limitations

- Module ownership means the actual declaring source module or namespace, not an inferred multi-file package or abstraction directory. Dedicated operation modules need an explicit ownership contract; proximity alone does not establish it.
- Structural equivalence does not establish shared ownership. Generic transformations can obscure declaration provenance. Inherited operations remain unresolved. Destination ownership follows the same declaration rules as the source; a destination wrapped in a library alias such as `Record` or `Omit` resolves to the library alias and is treated as unowned.
- Field reads need not be exclusive to a variant. The diagnostic identifies external representation construction, not proof that one owner is the best home for a conversion. Dedicated adapters are not recognized; exclude them with overrides.
- It does not follow receiver aliases, later uses of assigned results, destructuring, computed property access, or helper calls that conceal construction.
- Call arguments within a returned or assigned expression can supply construction evidence even when the call does not return that argument. Complete output data flow is not established.
- An if without an else is not a dispatch even when the following statements handle the remaining values. Predicates hiding tag tests, handler tables, and framework visitor routing are not analyzed.
- The rule does not detect duplicated conversions or prove that an operation exists only once.
- Diagnostic accuracy and generality remain unestablished. Do not treat passing tests or low finding counts as proof of readability improvement.
