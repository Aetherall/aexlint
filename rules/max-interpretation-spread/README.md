# max-interpretation-spread

Limits how many files check a literal union for particular values. Adding a value leaves each of those checks silently taking its fallback, so every file needs review.

```json
{ "rules": { "aexlint/max-interpretation-spread": ["warn", { "max": 4 }] } }
```

`max` is an optional positive integer, default 4.

## Example

Reported when five files of the program contain:

```ts
type Plan = "free" | "team" | "business";
declare const plan: Plan;
if (plan === "free") {
}
```

Accepted: a complete table makes a missing value a type error.

```ts
const LABELS: Record<Plan, string> = { free: "Free", team: "Team", business: "Business" };
```

## What counts

- A vocabulary is a union of at least two string or number literals (enum members included), ignoring `null` and `undefined`, read from the declared type, not the narrowed one. A vocabulary declared only in dependencies or declaration files, such as an AST node type, does not count: you interpret it but do not own it. [no-embedded-translation](../no-embedded-translation/README.md) asks where those interpretations belong.
- A check is a comparison with a literal, constant, enum member or single-literal property; `includes` on a literal array or tuple of its values; or a `switch` with a `default` or a missing value. Exhaustive switches, tables and `typeof` do not count.
- Vocabularies with the same set of values are one vocabulary. Each file counts once. The whole program counts, including unlinted files and tests, but not declaration files or `node_modules`.

## Reporting and limits

Above `max`, reports `spreadInterpretation` once per vocabulary in each linted file that checks it, on its first check. Help lists the other files. Suppressing a report does not reduce the count. No fixes.

Unrelated unions with the same values merge; subsets count separately. `in`, helper predicates and other tsconfig programs are not searched. Stable vocabularies and deliberate per-layer translations can legitimately exceed the limit.
