# no-lookup-fallback

Reports a lookup on a literal union that names most of its values and gives the rest one constant. Writing an answer for each value shows that each value needs its own answer, but the values left to the fallback, and every value added later, get the fallback's answer without anyone choosing it.

```json
{ "rules": { "aexlint/no-lookup-fallback": "warn" } }
```

No options.

## Example

Reported on `"ext"`, which also answers for `"visitor"`:

```ts
type Tier = "gold" | "silver" | "bronze" | "guest" | "visitor";
const badgeOf = (tier: Tier) =>
  tier === "gold" ? "gld" : tier === "silver" ? "slv" : tier === "bronze" ? "brz" : "ext";
```

Accepted: a complete table, where a new value is a type error.

```ts
const BADGES: Record<Tier, string> = {
  gold: "gld",
  silver: "slv",
  bronze: "brz",
  guest: "ext",
  visitor: "ext",
};
```

Also accepted: a default that covers most values (`size === "XS" ? "MINI" : "BODY"`), a fallback that uses the value or does work, and a `default` that throws.

## What counts

- A decision on a vocabulary, as in [no-repeated-value-group](../no-repeated-value-group/README.md#what-counts): a `switch`, an `if`/`else if` chain or run of `if`s, or a ternary chain testing one value. Vocabularies declared only in dependencies or declaration files are skipped: the project does not choose their values, and a default for values it does not know is often intended.
- Its fallback is the `default`, trailing `else` or last ternary alternative, or the `return` right after a `switch` without `default` or a run of `if`s whose branches all leave.
- At least two tested branches, each picking a constant or an empty value (`""`, `0`, `-1`, `null`, `undefined`, a boolean) without running anything. Tests must read only the tested value.
- The fallback does the same, and answers for at least two values, but fewer than the tested branches name.
- `null` and `undefined` are not values of the vocabulary.

## Reporting and limits

Reports `lookupFallback` on the fallback, naming the values it answers for. No fixes: listing the values, a table and an exhaustive `switch` are each a choice.

Generic parameters use their constraint; `any`, `unknown`, `never` and unresolved types have no vocabulary. Lookups through tables, maps or helper calls are not read. A fallback that should apply to future values, such as a platform default, needs a disable comment.
