# max-interpretation-spread

Limit how many files check a literal union for particular values. Adding a value leaves these checks silently taking their fallback path, so each file needs a manual review.

**Status:** experimental. Diagnostic-only; no fixes or suggestions.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/max-interpretation-spread": ["warn", { "max": 4 }] }
}
```

**Options:** optional `max`, a positive safe integer; default **4**. Unknown keys and invalid values are configuration errors. The default is not a validated universal limit.

See [typed-plugin setup](../../README.md#type-aware-setup-and-limits) for project requirements. Run `tsc` separately; editor integration is unsupported.

## Example

Reported when this check appears in five files of the same TypeScript program:

```ts
type Plan = "free" | "team" | "business";
declare const plan: Plan;
if (plan === "free") {
}
```

Accepted alternative: a complete table makes missing values a type error.

```ts
type Plan = "free" | "team" | "business";
const LABELS: Record<Plan, string> = {
  free: "Free",
  team: "Team",
  business: "Business",
};
```

## What counts

- A vocabulary is a union of at least two string or number literals, including enum members, after removing `null` and `undefined`. A generic's base constraint can qualify.
- The rule uses the declared type of an identifier, property read or call, not its flow-narrowed type. Parentheses and type wrappers are looked through.
- `===`, `!==`, `==` and `!=` count when one operand is a vocabulary and the other has a single literal type. Constants and enum members can be that operand.
- A `switch` counts if it has a `default` or omits a value. An exhaustive switch without `default` does not count. Table lookups and `typeof` comparisons do not count.
- Identity is the set of literal values, not the declaration. Copies with the same set are merged; enum members also retain their enum/member identity. Different sets are separate vocabularies.
- Each file counts once, however many checks it contains. The search covers project source files throughout the TypeScript program, including unlinted files and tests, but not declaration files or external libraries.

## Reporting scope

Above `max`, reports `spreadInterpretation` once per vocabulary in each linted file that checks it, on the value of its first check. Declaration-only files do not report.

The message gives the count, limit and local check count. Help names other files and declarations so you can judge whether the checks belong together. Suppressing a report does not reduce the count.

## Limitations

- Unrelated unions with identical literals are merged. Conversely, narrower subsets split one concept into several counts.
- Broad `string`, `boolean`, single literals and unions with nonliteral members do not qualify. Widened types hide checks.
- `includes`, `in`, helper predicates and `instanceof` are not recognized. The count is a lower bound, not a complete dependency map.
- Other tsconfig programs are not searched. Large programs can be expensive despite the shared syntax index.
- Stable vocabularies and deliberate per-layer translations may legitimately exceed the limit. A report does not establish that sharing a helper would improve the design.
