# max-expression-complexity

Limits the work packed into one expression.

```json
{ "rules": { "aexlint/max-expression-complexity": ["warn", { "max": 4 }] } }
```

`max` is a required positive integer.

## Example

Reported at `max: 4` (workload 5):

```ts
createResult(parseHeader(header), resolveOwner(owner), calculatePrice(items), permissions(user));
```

Accepted (final expression costs 3):

```ts
const headerValue = parseHeader(header);
const ownerValue = resolveOwner(owner);
createResult(headerValue, ownerValue, calculatePrice(items), permissions(user));
```

## Workload

- Calls, constructions, tagged templates and `import()` cost 1 plus their inputs. Binary operations, updates, compound assignments, `yield` and most unary operations cost 1.
- Reads, literals, plain assignment, `??`, `await`, `!` and `typeof` are free.
- A run of the same `&&`/`||` costs 1 plus its operands: `a && b && c` costs 1, `f() && g() && h()` costs 4.
- A ternary costs 1, plus its condition, plus its costlier alternative.
- Concatenation onto a known string is free.
- Arrays and JSX carry their costliest entry. A fluent method chain carries its costliest stage.
- Object fields, functions and classes are checked on their own and cost nothing to the enclosing expression.

## Reporting and limits

Reports `tooComplex` on the outermost over-limit region. No fixes: extraction can change evaluation order and short-circuiting. Callees and getters are not inspected. Other expression rules may report the same expression.
