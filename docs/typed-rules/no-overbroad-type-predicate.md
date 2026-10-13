# no-overbroad-type-predicate

Experimental, opt-in correctness rule. Requires type information and strict null checking. No fixes.

## Intent

A written `value is Target` predicate promises that a true result identifies `Target` **and that a false result excludes it**. A policy that accepts only some values permitted by `Target` overpromises on failure, even when its positive-side selection is useful.

This rule warns when a necessary comparison rejects a value allowed by the declared target. It does not inspect callers or claim an observed runtime bug. Positive-only filtering is an important usefulness counterexample; review findings before adopting this experimental rule.

Oxlint's `typescript/no-unnecessary-condition` checks redundant predicate calls and unnecessary conditions, not this relationship between a predicate's declared target and a rejecting body comparison. `typescript/no-unsafe-type-assertion`, `typescript/no-unsafe-return`, and `typescript/strict-boolean-expressions` do not establish that relationship either.

## Runtime

Native Go rule, run through `@aetherall/aexlint/typed-plugin` as `aexlint-typed/no-overbroad-type-predicate`. Enable it explicitly:

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/no-overbroad-type-predicate": "warn"
  }
}
```

## Reported code

```ts
type Product = { category: "physical" | "digital" };

function selected(value: Product | undefined): value is Product {
  return value?.category === "physical";
}
```

The warning covers `value?.category === "physical"`:

> This comparison rejects value?.category = "digital", which the declared predicate target "Product" permits; a false result does not exclude that target.

A `Product` with category `"digital"` can fail this check. A caller's false branch can nevertheless receive a type excluding `Product` because of the written predicate contract.

The rule reports **one diagnostic per predicate**, at its first supported rejecting comparison, even if several comparisons reject permitted values. Equality rejects a permitted alternative; inequality rejects its compared value when that value is permitted. For open `string`, the message uses a different permitted string, often `""`; this is a type-domain representative, not an observed runtime value or a whole-object witness.

Direct scalar comparisons, one direct property, optional access, presence-guarded access, reversed comparisons, and literal-typed constants are supported. Arrow functions, methods, and predicates about a later named parameter are supported too.

## Accepted code

Presence guards and appropriately narrowed targets do not have the detected mismatch:

```ts
type Product = { category: "physical" | "digital" };

function present(value: Product | undefined): value is Product {
  return value !== undefined;
}

function physical(value: Product | undefined): value is Product & { category: "physical" } {
  return value?.category === "physical";
}

function policy(value: Product): boolean {
  return value.category === "physical";
}
```

Boolean policies and inferred predicates are not checked. A written overbroad predicate still receives a warning when used only by `filter`: positive-only use does not change its declared negative promise. Returning boolean and narrowing the predicate target express different intentions; neither is an automatic repair, and narrowing just one field may leave other policy restrictions overbroad.

Imports and aliases use resolved compiler types. With unchanged predicate code:

```ts
import type { Product } from "./dependency.js";

export function selected(value: Product | undefined): value is Product {
  return value?.category === "physical";
}
```

A dependency declaring `{ category: "physical" }` produces no warning. Changing only that dependency to `{ category: "physical" | "digital" }` or `{ category: string }` produces one. Separate compiler programs are analyzed independently; this rule has no cross-program cache.

## Type information and supported boundary

The rule uses the compiler signature's predicate parameter index, original input type, resolved declared target, and property symbols from target members, not just the flow-narrowed type inside the body. The target must be assignable to the input.

Only synchronous nongenerator functions with a written identifier predicate and an expression body or a block containing one return are analyzed. All parameters must be named identifiers without defaults; the predicate parameter must not be rest. Destructuring and defaults on **any** parameter are conservatively excluded, even when harmless, because binding expressions and initializers can mutate the input before the returned comparison.

Known input and target types are required. `any`, `unknown`, `never`, unresolved/error types represented by `any`, and uninstantiated type parameters are skipped. Relevant property domains with those types are skipped too. Generic function declarations are excluded; concrete generic instances can be checked when the tested property has a concrete domain. Unrelated generic data fields are not inspected. Unknown-input validators are outside this bounded check, not certified sound.

Relevant target domains must be finite string/number literal sets or open `string`. Top-level `&&` comparisons can be necessary even with unsupported conjuncts: a supported pure prefix may precede a restriction, and an opaque suffix does not invalidate an earlier rejection. Calls, assignments, accessors, casts, or other unsupported potentially effectful prefixes block evidence after them. A top-level `||` is not analyzed as a necessary comparison.

Getter/setter and auto-accessor reads are excluded whenever any resolved property declaration is an accessor, including declarations hidden in merged symbols on unions/intersections and optional reads. For example, a getter in `value.ready && value.category === "physical"` can normalize `category` before it is tested, so the later comparison is not evidence of rejection. Reversing the conjunction can still report the earlier data-property comparison: a rejected value never reaches the getter. Ordinary data-property prefixes remain supported.

## Options

None. Omitted options, `null`, and `{}` are accepted; other values and unknown fields are rejected in Go before analysis.

`strictNullChecks: true` or `strict: true` is required. Explicitly disabling strict null checking reports `strictNullChecksRequired` at the start of each linted file instead of analyzing predicates.

## Fixes

None. The policy itself may be intentional. Review the negative promise and choose a contract matching the selection's intent; no fix or suggestion is emitted.

## Limitations

Declared property domains can be wider than reachable runtime values. The rule does not construct complete object witnesses, prove correlated field, brand, class, or constructor invariants, or model hidden getters/proxies behind structural interfaces. It does not prove the positive promise or complete predicate soundness.

Unsupported cases include multi-statement bodies, early-return ladders, nested/element property comparisons, arbitrary computed constants, broad number/boolean domains, assertions, `this` predicates/parameters, and nullish rejection as an independently reported value. A skipped predicate is not evidence of correctness.

Requires a TypeScript 7-compatible `tsconfig.json` project with dependencies installed. TypeScript errors remain the compiler's responsibility; run `tsc` too. No editor integration is supported.
