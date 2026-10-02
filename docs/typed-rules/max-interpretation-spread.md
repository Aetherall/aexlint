# max-interpretation-spread

## Intent

Limit how many files check a closed vocabulary for particular values. A closed vocabulary is a type made only of string or number literals, such as `"free" | "team" | "business" | "enterprise"`. A check like `plan === "free"`, or a `switch` that skips values or has a `default`, singles out some values and treats the rest as "everything else". When a value is added, the compiler flags none of these checks: each one silently sends the new value to its "everything else" branch. Every file that makes one must be found and reviewed by hand. The cost grows with the number of files, not the number of checks: checks concentrated in one designated place, such as an interpreter or serializer module, count as one file.

The vocabulary is identified by its literal values, not by where it is declared. Redeclared copies (a type alias, a class member typed inline with the same literals, a versioned serializer's own copy) are one vocabulary, because adding a value means updating each copy and each check.

The diagnostic describes the spread. It does not prescribe a remedy. An exhaustive `switch`, a `Record<Vocabulary, T>` table, a predicate owned by the vocabulary's home, or one shared translation are all ways to make the compiler point to the sites again or to reduce their number.

Status: experimental.

Oxlint 1.83.0 has no rule that measures checks across files. `typescript/switch-exhaustiveness-check` is complementary: it turns a single `switch` into a site the compiler points to, which this rule then no longer counts.

## Runtime and options

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/max-interpretation-spread": ["warn", { "max": 6 }]
  }
}
```

`max` is optional and must be a positive safe integer: the greatest number of files, including the linted one, that may check one vocabulary for particular values. It defaults to 4, also when the rule is enabled without options (`"warn"`). Unknown properties, non-object options, and non-integer or non-positive values produce a backend configuration error. The default is not a validated universal boundary: stable vocabularies and deliberate per-layer translations can justify a higher limit.

## Measurement

**Values.** A property read, identifier (local, parameter, destructured binding), or call, whose declared type (not the flow-narrowed type), with `null` and `undefined` removed, is a union of at least two string or number literal types, enum members included. A type that is not such a union but whose base constraint is (a type parameter `K extends Kind`) qualifies through its constraint. `boolean`, a single literal, and unions containing `string`, objects, or other non-literals do not qualify. Parentheses, `as`, `satisfies`, non-null assertions, and type assertions are looked through.

**Vocabulary.** The sorted set of literal values. Enum members are identified by their enum and member as well as their value, so `Level.High` and the number `1` differ.

**Checks.** Subset selections the compiler does not point to when a value is added:

- `===`, `!==`, `==`, or `!=` between a value and an operand with a single string or number literal type (a literal, a `const`, an enum member, an `as const` object member);
- a `switch` on a value that has a `default` clause or does not list every value.

Not counted: exhaustive `switch` statements without `default`, element access (`TABLE[value]`, which a `Record<Vocabulary, T>` makes exhaustive), comparisons between two vocabulary values, and `typeof` comparisons.

**Count.** The number of distinct files with at least one check on the vocabulary, searched across the whole TypeScript program containing the linted file, including files that are not being linted. The result does not depend on which files are selected for linting. Test files that belong to the program count like any other file.

## Reported code

```ts
// kinds.ts
export type Plan = "free" | "team" | "business" | "enterprise";

// billing.ts, invoices.ts, limits.ts, settings.ts, emails.ts
if (account.plan === "free") {
}
```

With the default limit of 4, five files check `Plan`. Each of the five files reports `spreadInterpretation` once per vocabulary, on the value of its first check (`account.plan`). The linted file must itself make a check: a file that only declares or passes the vocabulary is not reported.

The message names the vocabulary, lists its values (the first four and the count beyond five), gives the file count and the limit, states that the compiler flags none of these checks when a value is added, and counts the checks in the linted file:

```text
`Plan` ("business" | "enterprise" | "free" | "team") is checked for particular values in 5 files (maximum 4). The compiler points to none of these checks when a value is added. This file has 1 such check.
```

The help lists up to three other files, nearest first (most shared leading directories, then path), and up to four declarations the checked values come from, with their files:

```text
Other files, nearest first: emails.ts, invoices.ts, limits.ts, and 1 more. Declared as: Plan (kinds.ts).
```

Declarations are, in order of preference for each check: the type alias or enum on the value's declared type; otherwise the property read, qualified by its enclosing type alias, interface, or class (one entry per declaration when read through a union of object types); otherwise the variable, parameter, or function. They are ordered by how many checks read them, then aliases and enums before properties. The vocabulary's name in the message is the first declaration's name. Paths are relative to Oxlint's working directory. When the message elides values, the help ends with the full list (`Values: ...`), so vocabularies that differ only beyond the fourth value can be told apart. Oxlint plugins have no help field, so the help follows the message on a line starting with `help: `.

## Accepted code

```ts
// One file: many checks count once.
switch (plan) {
  case "free":
    return 1;
  default:
    return 0;
}
if (plan === "business") {
}

// Exhaustive: the compiler points here when a value is added.
switch (plan) {
  case "free":
  case "team":
  case "business":
  case "enterprise":
    return 1;
}
const LABELS: Record<Plan, string> = {
  free: "Free",
  team: "Team",
  business: "Business",
  enterprise: "Enterprise",
};
LABELS[plan];
```

Not counted:

- `a.plan === b.plan`, because neither operand is a single literal type;
- `label === "x"` when `label: string`, and `enabled === true` when `enabled: boolean`;
- reads that are not checks, such as `return account.plan` or `format(account.plan)`;
- `typeof value === "string"`.

A different literal set is a different vocabulary, even one that contains this set: `"a" | "b" | "c" | "d"` checks do not count for `"a" | "b" | "c"`.

Cross-file example: the linted file compares `kind === "a"` where `kind: Kind` is imported from `kinds.ts`, and another file does the same. With `export type Kind = "a" | "b"`, the file is reported at `{ "max": 1 }`. Changing only `kinds.ts` to `export type Kind = string` removes the report.

## Fixes

Diagnostic-only. No fixes or suggestions are emitted. Reducing spread requires choosing where the interpretation belongs, which the rule cannot determine.

## Limitations

- **Coincidences.** Unrelated vocabularies with the same literals are merged: `"left" | "right"` for message alignment and for sidebar handles, `"small" | "medium" | "large"`, version numbers `1 | 2`. The declarations in the help show when this happens.
- **Splits.** A large discriminated union read through narrower union types is several vocabularies, one per literal set. The count undercounts such concepts.
- **The count is a lower bound.** Not recognized: `[...].includes(value)`, the `in` operator, predicates taking a vocabulary literal (`value.is("a")`), passing the value to a helper that branches, and `instanceof` checks over class hierarchies.
- **Declared types are trusted.** Checks on values whose types were widened to `string` are not counted.
- **Scope is the linted file's TypeScript program.** Other tsconfig projects that check the same vocabulary, such as applications referencing a shared library, are not searched. Shared code linted as part of several projects is counted in the program headless mode assigns it to.
- **One report per file.** Every file over the limit is reported, so one vocabulary produces as many diagnostics as it has files. Silencing one of them does not reduce the count.
- **Cost grows quadratically with program size.** The first linted file in a program builds a syntactic index of every equality and `switch`, shared by all invocations ([program-wide indexes](../native.md#program-wide-indexes)). An invocation whose file has no check returns after classifying its own branches. Otherwise it classifies, with its own checker, the indexed branches that compare against one of its vocabularies' literal values, plus every branch that compares against something other than a literal token.
- **Spread is not proof of a problem.** Stable vocabularies that never change, and deliberate per-layer translations, can exceed any limit legitimately.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`. Editor integration is not supported yet.
