# prefer-domain-predicate

## Intent and status

Experimental diagnostic: identify reconstructed classifications that have no explicit name. A named result, a named category of alternatives, or an encapsulating predicate/validator can supply that name. Prefer an existing operation such as `node.isContainer()` when appropriate; a one-off check does not require a helper just to satisfy the rule.

This is not a membership-syntax ban. The detector groups checks by the subject inspected, across comparisons, ranges, and collection membership. Naming a category satisfies a standalone membership check; combining it with other representation checks may introduce another concept that needs its own name.

Oxlint's `unicorn/prefer-includes` and `typescript/prefer-includes` prefer membership syntax over index comparisons; `eslint/no-magic-numbers` targets numeric constants. None identifies a classification reconstructed from multiple checks of the same subject.

## Runtime and configuration

Syntax/scope-only Oxlint JS plugin. No compiler types or guessed domain method names. No options, fixes, or suggestions.

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/prefer-domain-predicate": "warn" }
}
```

Run `pnpm lint:classifications` to inspect aexlint itself. This run is separate from mandatory `pnpm lint`; findings need assessment before adoption. There is no recommended preset.

## Reported code

```ts
if (node.type === "ArrayExpression" || node.type === "JSXElement") use(node);
if (response.status >= 200 && response.status < 300) consume(response);
if (user.role === "owner" || user.permissions.includes("manage")) edit();
if (["ArrayExpression", "JSXElement"].includes(node.type)) use(node);

const containerTypes = ["ArrayExpression", "JSXElement"];
if (containerTypes.includes(node.type) && node.state === "ready") use(node);
```

Reports `preferPredicate` on the outermost classification expression, not each comparison. The message identifies the inspected subject. Several subjects in one expression share one diagnostic; nested expressions already covered by that diagnostic are suppressed.

## Evidence counted

The initial detector requires at least two distinct representation facts about the same subject within one expression:

- Equality/inequality and ordered comparisons against string, numeric, or bigint literals. Signed numeric literals and transparent TypeScript wrappers are recognized; reversed comparisons are normalized.
- `includes`/`has` calls over literal arrays or unshadowed `new Set([...])`: each distinct literal alternative contributes a fact about the argument.
- A named collection (an identifier or static member path) contributes one category fact, not one fact per entry. It passes alone, but can combine with another check of the same subject. Local, imported, parameter, mutable, and aliased collection names are treated alike; their definitions are not expanded. `new Set(namedCategory)` retains that naming boundary.
- Membership of a fixed literal in an inspected property, such as `user.permissions.includes("manage")`, contributes one fact about its owner. It can combine with other checks such as `user.role === "owner"`.
- AND/OR groups and negation combine facts. Parentheses, optional chains, and TypeScript wrappers preserve them. Duplicate comparisons and duplicate collection entries do not increase evidence.

Subjects are identifiers, `this`, and static member paths. A comparison of `response.status` inspects `response`; membership in `user.permissions` inspects `user`. For nested paths the immediate owner is used: `node.parent.type` inspects `node.parent`, not `node`. Computed literal keys are supported; dynamic keys and call-produced subjects are not matched.

No facts are combined across statements, function boundaries, ternary alternatives, or arbitrary calls. Null/undefined, booleans, `typeof`, property-existence guards, and already-named flags are not classification facts. This deliberately avoids many structural validation idioms rather than estimating their domain meaning.

## Accepted code

```ts
if (node.isContainer()) use(node);
if (isContainer(node)) use(node);

const isX = a.amount > 10 && a.type === "money";
const expensiveMoney = a.amount > 10 && a.type === "money";
if (expensiveMoney) use(a);

const containerTypes = ["ArrayExpression", "JSXElement"];
if (containerTypes.includes(node.type)) use(node);

function isContainer(node) {
  return node.type === "ArrayExpression" || node.type === "JSXElement";
}

const isSuccessful = (response) => response.status >= 200 && response.status < 300;

class Node {
  isContainer() {
    return ["ArrayExpression", "JSXElement"].includes(this.type);
  }
}

if (left.type === "first" || right.type === "second") use(left, right);
if (value != null && typeof value === "object") inspect(value);
if (ready && permitted && connected) proceed();
```

### Named results and categories

A directly named result qualifies without an `is` prefix: a simple variable initializer (`const`, `let`, or `var`), a plain assignment to an identifier or static member path, or a statically named object/class field. Parentheses, TypeScript wrappers, negation, and logical composition preserve this boundary. The rule does not prove that the expression has boolean type or that the name is good.

The name must belong to the classification itself. `const label = classification ? 'yes' : 'no'`, `const result = send(classification)`, and `const items = source.filter(item => classification)` still report the anonymous classification. Naming the larger computation does not name its embedded decision. Dynamic property names, destructuring, and compound assignments do not qualify.

A named category is one semantic unit. `allowed.includes(node.type)` passes, but `allowed.includes(node.type) && node.state === 'ready'` and `allowed.includes(node.type) && privileged.includes(node.type)` still report unless their combined result is named. Duplicate uses of the same category and subject count only once. A member read such as `user.permissions.includes('manage')` still combines with `user.role === 'owner'`; the role-or-permission policy remains anonymous.

### Named operations

A named predicate boundary is recognized structurally, not by an `is`/`has` naming prefix: an expression-bodied named/bound function, or a named function with a terminal return statement. Before that return, local variable declarations and early guards returning literal `true` or `false` are allowed. This permits meaningful intermediates and missing-value guards without penalizing the returned classification. It is not a purity analysis.

Static object/class methods, getters, function-valued fields, and named assignments also qualify. The classification must contribute directly to the returned result through logical/negation/transparent wrappers. A check in an `if`, a call argument, or a ternary condition inside an arbitrary named function remains reportable.

Anonymous callbacks are not named predicate boundaries, but named results inside them are allowed. More general control flow and indirect returns need explicit intermediate naming or remain outside the recognized predicate shapes.

## Validation boundaries

A named validation-only function or method is also a legitimate owner of representation checks: it enforces a concept instead of answering a question. The exemption depends on the whole body, not the function name, return annotation, or the presence of a nearby throw.

```ts
class Note {
  checkIntegrity(): void {
    if (this.value < 0 || this.value > 10) {
      throw new RangeError("Expected a value between 0 and 10");
    }
  }
}
```

The initial structural recognizer accepts only:

- Top-level guards without `else`, whose body is exactly a throw or a bare return. A block around that one statement is allowed. At least one guard must throw.
- Optional terminal `return;`, or normal fallthrough. Returning a value, even `undefined`, does not qualify.
- Simple `const` bindings with read-only-shaped initializers: literals, identifiers, property access, operators, templates, arrays without spread, and the membership operations already recognized by this rule.
- The same read-only-shaped expressions in guard conditions. Calls other than recognized membership tests, mutation, assignments, destructuring, loops, and unrelated statements prevent the exemption. Error construction/factories in the throw expression are allowed.

Only classifications directly composing a guard condition or a local initializer are exempted by validation ownership. Classifications hidden inside error-factory arguments or nested anonymous callbacks retain their own diagnostics. A named boolean intermediate can independently satisfy the named-result boundary even when the surrounding function is not a validator.

```ts
function editDocument(doc) {
  if (doc.state === "locked" || doc.state === "archived") {
    throw new AccessDenied();
  }
  performEdit(doc);
}
```

This still reports: the function interprets the policy and also performs business work. The exemption is intentionally conservative; delegated validators and more complicated validation control flow may still report. Getter effects, coercion hooks, and custom implementations of collection methods cannot be proven absent by this syntax-only recognizer.

## Limitations and interpretation

The rule cannot prove a domain concept exists, that an available API implements it, or that extraction is desirable. Numeric input bounds can look exactly like a meaningful domain range. Those findings need human review. A poorly named helper can satisfy the structural boundary; name quality remains a human judgment.

Method spellings `includes` and `has` are syntactic evidence, not proof of standard collection semantics. Collection contents and aliases are not evaluated across bindings. Lexical scope is used to reject shadowed `Set` constructors. Arbitrary helper implementations, switches, regexes, boolean fields, and cross-statement classifications are outside the initial analysis. The rule neither resolves an existing domain predicate nor proposes a name.

No automatic fix is safe: extraction can affect evaluation order, getters, short-circuiting, scope, and receiver binding. Avoid moving logic merely to lower a score. This rule is independent of the complexity rules and does not alter their formulas or limits.
