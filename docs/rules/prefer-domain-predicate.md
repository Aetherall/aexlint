# prefer-domain-predicate

Experimental, syntax/scope-only Oxlint JS diagnostic. Finds classifications rebuilt from multiple checks of one subject without a name of their own. A named result, category, predicate or validator can supply that name; a one-off check does not require a helper.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/prefer-domain-predicate": "warn" }
}
```

No options, fixes, suggestions or recommended preset. `pnpm lint:classifications` evaluates the rule separately from mandatory self-lint. Review findings before adopting it.

## Example

Reported:

```ts
if (node.type === "ArrayExpression" || node.type === "JSXElement") use(node);
```

Accepted:

```ts
const isContainer = node.type === "ArrayExpression" || node.type === "JSXElement";
if (isContainer) use(node);
```

## What counts

At least two distinct representation facts about the same subject must occur within one expression:

- Equality, inequality or ordered comparisons against string, numeric or bigint literals. Signed numbers, literal templates, reversed comparisons and TypeScript wrappers are recognized.
- Single-argument `includes`/`has` calls on literal arrays or unshadowed `new Set([...])`. Each distinct literal alternative contributes a fact; holes, spreads and nonliteral entries are not expanded.
- A named collection (identifier or static member path) contributes one category fact, regardless of its contents or origin. `new Set(namedCategory)` preserves that boundary. Definitions and aliases are not expanded.
- Membership of a literal in an inspected property contributes a fact about its owner: `user.permissions.includes("manage")` combines with `user.role === "owner"`.

AND/OR and negation combine facts; parentheses, optional chains and TypeScript wrappers preserve them. Duplicate facts do not count twice. Subjects are identifiers, `this` or static member paths, including computed literal keys. A property check inspects its immediate owner: `node.parent.type` inspects `node.parent`, not `node`. Dynamic keys and call-produced subjects are not matched.

Facts are not combined across statements, functions, ternary alternatives or arbitrary calls. Null/undefined, booleans, `typeof`, property-existence checks and named flags are not facts.

## What supplies a name

A directly named result qualifies without an `is` prefix: a simple variable initializer, plain assignment to an identifier/static member path, or statically named object/class field. Logical composition, negation and transparent wrappers preserve the name. Destructuring, dynamic keys and compound assignments do not qualify.

Naming a larger computation does not name its embedded classification. A classification inside `send(classification)`, a ternary condition or an anonymous `filter` callback still reports even when the overall result is assigned to a variable.

A named category passes alone. `allowed.includes(node.type) && node.state === "ready"`, or combining two different categories for that subject, still needs a name for the combined result.

A named/bound predicate may directly return the classification through an expression body or terminal return. Before that return, only variable declarations and no-`else` guards returning literal `true` or `false` are allowed. Static methods, getters, function-valued fields and named assignments qualify. Checks in call arguments, ternary conditions or other control flow do not inherit the function's name. Anonymous callbacks are not named predicates, but may contain named results.

A named validation-only function also qualifies when its whole body contains only:

- Top-level no-`else` guards whose body is exactly a throw or bare return, optionally in a block. At least one guard must throw.
- Simple `const` bindings and guard conditions using read-only-shaped literals, identifiers, property reads, operators, templates, arrays without spread or recognized membership calls on readable receivers.
- An optional terminal `return;` or fallthrough, never a returned value.

Mutation, destructuring, loops, unrelated statements and other calls prevent validation ownership. Error construction/factory calls in throws are allowed. Only classifications directly composing guard conditions or local initializers are exempt; checks hidden in error arguments or anonymous callbacks still report. A function that checks then performs business work is not validation-only.

## Reporting and limits

Reports `preferPredicate` on the outermost classification expression, naming its subjects. Multiple subjects share one report; covered nested classifications are suppressed. Independent expressions report separately.

The rule cannot establish domain meaning, boolean types, name quality, purity or standard collection semantics. Numeric bounds may resemble domain ranges; getters and custom `includes`/`has` methods may have effects. Switches, regexes, arbitrary helpers and cross-statement classifications are outside its analysis. It neither resolves existing predicates nor proposes names.

Extraction can change evaluation order, short-circuiting, scope and receiver binding. Review the concept, not just the warning; complexity-rule scores and limits are unaffected.
