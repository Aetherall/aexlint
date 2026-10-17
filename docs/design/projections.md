# Interpretations belong in projections

**Status: implemented as [no-embedded-translation](../../rules/no-embedded-translation/README.md), under evaluation.**

## Principle

Code often turns a general value into its own concept. A lint rule turns an AST node into "an invocation with inputs". An app turns an API response into "an order that can be refunded".

That translation depends only on the value's content. It should live in one place: with the value's owner, or in a local projection. When it is written inline in the logic that uses it, the logic reads as mechanics, and every change to the value's variants has to be found in that logic.

## Two kinds of translation

- **Extraction** turns a value into data: the callee and inputs of a call, the name of a declaration.
- **Decision** turns a value into a category: is a boundary, is an exit.

Both read only the value. Neither reads the caller's state.

## Where it belongs

- A variant the project owns: its owner, as a method or a helper next to it. `no-fact-outside-owner` and `no-external-variant-behavior` cover this.
- A variant from a dependency (an AST, an API, DOM events): a local **projection**. This is a function from the value to the project's own model, or `undefined`, for example `invocation(node) → { receiver, inputs } | undefined`. A table keyed by variant is the same idea.

The logic then works on the project's model and does not test variants itself. A dependency upgrade means reviewing the projections only.

## Telling them apart

The decision model records, for each branch, what its result depends on and whether it runs commands (see [outcome](decision-model.md)). A pilot on `max-expression-depth` showed that this alone does not separate the cases: an inline switch extracting a node's parts looks like a projection, and a projection reading a second parameter looks mixed. What separates them is where the result goes:

- **Embedded translation:** a tested branch reads the value's parts and keeps the result in a local of its function. The function goes on to do other work with it.
- **Projection:** the tested branches return their result.
- **Dispatch to a handler:** a branch passes the whole value on. This is not a translation.

This uses data flow and types only, no names.

## Relation to existing rules

`no-external-variant-behavior`, `no-fact-outside-owner`, `max-interpretation-spread` and `no-procedural-value-dispatch` look like facets of this principle. For dependency-owned variants, `max-interpretation-spread` used to count projections and embedded translations alike. It now skips vocabularies declared only in dependencies, and `no-embedded-translation` covers them.

## Risks

- A projection used once adds indirection. The trigger is mixing the value's parts with other state, not every variant test.
- Framework dispatch, such as Oxlint selectors keyed by node type, is already an inversion.
- Building small objects in hot paths has a cost.
- A projection over a large external union is not complete. One explicit place to review is the point.

## Next steps

1. Evidence: fresh findings: 2 smell, 1 partly, 2 fine of 5, no false statement.
