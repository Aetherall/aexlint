# no-overbroad-type-predicate

Reports a `value is Target` predicate whose body rejects values `Target` permits. A false result promises the value is not `Target`, so callers' false branches get a wrong type.

```json
{ "rules": { "aexlint/no-overbroad-type-predicate": "warn" } }
```

No options. Requires `strictNullChecks`; without it, reports `strictNullChecksRequired` at the start of the file.

## Example

Reported on the comparison, which rejects `category = "digital"`:

```ts
type Product = { category: "physical" | "digital" };
function selected(value: Product | undefined): value is Product {
  return value?.category === "physical";
}
```

Accepted: a presence test, a narrowed target such as `value is Product & { category: "physical" }`, or a plain `boolean` result.

## What counts

- Synchronous, non-generic functions, methods and arrows with a written predicate, whose body is one returned expression. Parameters must be plain identifiers without defaults.
- A necessary comparison, alone or in a top-level `&&`, of the parameter or one of its properties (optionally through `?.` or a presence guard) with a literal or literal constant.
- The tested domain in the target is a finite set of string or number literals, or `string`. Accessor properties, `any`, `unknown`, `never` and unresolved types are skipped.

## Reporting and limits

Reports `overbroadTypePredicate` once per predicate, at its first rejecting comparison, naming a permitted value it rejects (for `string`, a representative such as `""`). No fixes: returning `boolean` and narrowing the target express different intents.

Declared domains can be wider than reachable values. Multi-statement bodies, `||`, nested properties, `this` predicates and assertion functions are not analyzed.
