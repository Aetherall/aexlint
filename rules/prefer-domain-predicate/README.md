# prefer-domain-predicate

Reports classifications rebuilt from several checks of one subject without a name of their own. A named result, collection, predicate or validator supplies the name; a one-off check does not need a helper.

```json
{ "rules": { "aexlint/prefer-domain-predicate": "warn" } }
```

No options.

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

- At least two distinct facts about one subject in one expression. A fact is a comparison with a string, number or bigint literal, each literal of `includes`/`has` on a literal array or `new Set([...])`, a named collection (one fact), or membership of a literal in a property of the subject.
- Subjects are identifiers, `this` and static member paths; `node.parent.type` is about `node.parent`.
- Nullish, boolean, `typeof` and property-existence checks are not facts. Facts are not combined across statements, functions or ternary alternatives.

## What names it

- A variable initializer, assignment to a static path, or a named field holding the classification.
- A named function, method, getter or field that returns the classification directly, after only declarations and guards returning `true`/`false`.
- A named validation-only function: top-level guards that throw or return, at least one throwing, and simple `const` bindings.

Naming a larger computation does not name a classification inside it, such as one passed to a call or used as a ternary condition.

## Reporting and limits

Reports `preferPredicate` on the outermost classification, naming its subjects. No fixes. The rule cannot judge domain meaning or name quality, does not find existing predicates, and ignores switches and regexes.
