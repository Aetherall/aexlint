# prefer-truthy-presence-check

## Intent

Prefer `if (value)` or `if (!value)` over an explicit comparison with `undefined` when the resolved type proves those boolean tests equivalent. This is an opt-in style rule, not a correctness error.

Oxlint 1.83.0's `typescript/no-unnecessary-condition` detects conditions whose result is already determined; this rule handles meaningful presence tests that still have two possible outcomes. `typescript/no-unnecessary-boolean-literal-compare` concerns comparisons with boolean literals, not `undefined`. `typescript/strict-boolean-expressions` controls which types may appear as conditions; configure it to permit the nullable types you choose to test by truthiness, or do not enable this preference alongside a conflicting policy.

## Runtime and configuration

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/prefer-truthy-presence-check": "warn"
  }
}
```

Requires `strictNullChecks` (including through `strict: true`). If disabled, reports `strictNullChecksRequired` at the start of the file and performs no comparison analysis.

## Reported code

```ts
declare const entry: { id: string } | undefined;
if (entry !== undefined) {
}
if (undefined === entry) {
}
```

Use `if (entry)` and `if (!entry)`, respectively.

Reports `preferTruthyCheck` for `!=` and `!==`, and `preferFalsyCheck` for `==` and `===`, spanning the complete binary comparison, excluding surrounding parentheses. Either operand order is supported. Parentheses around either operand are allowed.

Only boolean-consumed comparisons are eligible: `if`, `while`, `do/while`, `for` conditions, ternary conditions, and operands of logical negation. `&&` and `||` propagate eligibility only when their resulting value is itself consumed as a boolean. No descent through calls, assignments, `??`, casts, or other value-producing contexts.

## Type information

Use the actual flow-sensitive type at the compared value, resolving a generic's base constraint when available. The identifier spelled `undefined` must itself have the exact `undefined` type; a shadow with another type is not the sentinel.

The value must contain an `undefined` union member and at least one present member. For strict comparisons, a `null` member prevents reporting. Loose comparisons also remove `null` before assessing the present members.

Supported present types:

- Nonempty string literals, nonzero finite numeric literals, nonzero bigint literals, and the literal `true`.
- Symbols.
- Object types that do not admit falsy primitives. Structural types such as `{}`, `Object`, and `{ toString(): string }` do not establish truthiness just because the checker labels them object types.

When the global DOM `HTMLAllCollection` type exists, any object type admitting it is excluded: `document.all` is a legacy falsy object. This deliberately excludes broad types such as `object` in DOM projects and some structurally compatible callable types.

Every present union member must qualify. Broad `string`, `number`, `bigint`, and `boolean` types, `any`, `unknown`, `never`, unresolved types, enums, intersections, and unevaluated template/conditional types are not treated as proof of truthiness. A constrained generic qualifies only if its resolved constraint satisfies the same checks. Flow narrowing can make an otherwise broad type eligible, or eliminate `undefined` and put the comparison outside this rule's remit.

## Accepted code

```ts
declare const text: string | undefined;
if (text !== undefined) {
}

declare const entry: { id: string } | null | undefined;
if (entry !== undefined) {
}

declare const optional: { id: string } | undefined;
const present = optional !== undefined;
const result = optional !== undefined && optional;
```

An empty string is falsy but present. Strict comparison distinguishes `null` from `undefined`. Boolean-valued assignments and returned logical expressions must retain their value semantics, even when a surrounding TypeScript annotation says `boolean`.

For a cross-file example, keep `import { value } from './dependency.js'; if (value !== undefined) {}` unchanged. Exporting `value` as `{ id: string } | undefined` reports; changing only the dependency's type to `string | undefined` does not.

## Options

No configurable options. Omitted options, `null`, and an empty object are accepted. Other values and unknown keys produce a backend configuration error rather than being ignored.

## Fixes

Diagnostic-only: no fixes or suggestions are emitted. The typed plugin does not forward fixes. The diagnostic distinguishes a truthy test from a negated/falsy test without rewriting surrounding expressions or comments.

## Limitations

Type declarations, assertions, and compiler flow analysis are trusted; this is not a runtime validator. Unchecked indexed access can omit `undefined` from a type, in which case no presence preference is reported. Unsupported types are conservatively accepted rather than guessed at.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`; where a type cannot be resolved the rule does not report. Editor integration is not supported yet.
