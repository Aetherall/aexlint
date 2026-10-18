# Rule pairings

Candidate rules that complete the existing ones. Not shipped; each needs a contract, tests and corpus evidence first.

## Principle

The rules ask that what code knows about a value is stated once, where the value is owned. Typed rules sort by how that knowledge goes wrong:

| How the fact goes wrong          | Rules                                                                                                                                                                             |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Copied across files              | `max-interpretation-spread`, `no-repeated-value-group`, `no-repeated-value-policy`, `no-repeated-conversion`, `max-projection-spread`                                             |
| Established outside its owner    | `no-fact-outside-owner`, `no-external-variant-behavior`, `no-embedded-translation`                                                                                                |
| Lost or doubled at a boundary    | `no-repeated-presence-check`, `no-caller-enforced-invariant` (caller to callee), `no-forgeable-validated-result` (check to result), `no-repeated-selection` (selector to handler) |
| The test itself is wrong or deep | `no-overbroad-type-predicate`, `no-procedural-value-dispatch`                                                                                                                     |

Syntax rules limit what a reader holds at one point: one expression (`max-expression-depth`, `max-expression-complexity`, `max-decision-depth`, `max-call-assembly`) or one nesting chain (`max-held-context`, `no-else-if`, `no-multiline-condition`). `prefer-domain-predicate` is the bridge: name it instead of holding it.

## Candidates

### 1. A fallback absorbing a closed union

Started as [no-lookup-fallback](../../rules/no-lookup-fallback/README.md), narrowed to lookups: on the corpora, fallbacks after branches that do work were mostly intended selections.

A decision on a vocabulary ends in an untested arm (`default`, trailing `else`, the false side of a ternary chain) that covers values no test names and does no work: it returns `""`, `undefined`, `-1` or a constant. A new value compiles and silently takes the fallback.

- Pairs with `max-interpretation-spread`, which counts files holding such checks but not which one hides a value, and with `no-overbroad-type-predicate`, its mirror: that one rejects too much, this one accepts too much silently.
- Oxlint's `typescript/switch-exhaustiveness-check` covers `switch` only. Nothing covers if chains, ternaries or constant fallbacks.
- Evidence: 168 dispatches on the corpora whose fallback covers two or more values.
- Model fit: decisions already record the untested arm, its claim and its outcome.

### 2. A closed union widened to `string`

Started as [no-widened-union](../../rules/no-widened-union/README.md), for parameters and results. Measured first: the `status` and environment sites cited below are `string` from their source (parsed JSON, configuration), so no union is lost there and the rule does not report them.

A parameter or value typed `string` is compared with literals while its producers pass a union. Typo and completeness checks are lost, and the model's value rules cannot see the site at all ("plain `string` values get no vocabulary guessed").

- Pairs with every vocabulary rule: it reports where the type was thrown away, which makes those sites visible to the others again.
- Evidence: about 15 to 20 sites, including a CI pipeline `status` whose consumers disagree on `canceled`.
- Needs call sites across files: boundaries plus a shared index.

### 3. Narrowing lost on return

A function tests a value (`x.kind === "a"`) and returns it typed as the wide union; its callers test it again.

- The callee-to-caller direction of a boundary; `no-repeated-selection` covers only selector and handler gates.
- The model follows returns only to call sites in the same file.
- Measure first: literal rechecks in the other direction found no new findings.

### 4. Values held across statements

A syntax rule counting locals declared earlier and still read later at each statement: what the reader carries between statements.

- Every syntax rule today measures inside one expression or one nesting chain. `max-call-assembly` counts outside names meeting at one call only.
- Oxlint's `max-statements` and `complexity` measure other things.

## Already dropped

Conflicting lookups, constant lookups outside an owner, duplicate owner writes and repeated admission policy.
