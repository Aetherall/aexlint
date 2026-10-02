# max-held-context

## Status

Prototype. The measurement below is a first contract; its boundaries are provisional and no limit is recommended.

## Intent

Limit how many enclosing contexts a reader must keep in mind at the point where a decision is made: branches, loop bodies, `switch` cases, `catch` handlers, ternary alternatives, and callbacks passed into another operation. This is decision nesting measured across statements and across callback boundaries, with callback context counted only where a decision is made inside it.

Length is not measured. A long procedure of flat steps, guard clauses, and uniform `else if` chains is accepted however many decisions it contains.

Oxlint 1.83.0 does not measure this:

- `max-depth` counts block nesting within one function and restarts inside every callback, so a loop, a branch, a callback, and a guard inside it is never reported.
- `max-nested-callbacks` counts callback nesting alone, so callbacks that decide nothing are reported.
- `complexity` and totals such as Cognitive Complexity grow with every step, including independent guards and uniform `switch` cases.

## Runtime and options

Oxlint JavaScript plugin; syntax only, with no TypeScript type information.

```json
{
  "jsPlugins": ["aexlint"],
  "rules": {
    "aexlint/max-held-context": ["warn", { "max": 3 }]
  }
}
```

`max` is required and must be a positive safe integer. Exactly one options object is required; unknown properties are rejected. There is no default or recommended threshold. The value 3 illustrates configuration only.

## Measurement

**Measured units.** The program, and every function that is not passed directly as an argument to a call or construction, is measured from zero: function declarations, methods, object-property and variable-initialized functions, returned functions. Class bodies and their members are separate units. A function passed as an argument belongs to the unit that passes it.

**Contexts.** Each of these adds one open context for the code inside it:

- the consequent of an `if`, and its alternate unless the alternate is itself an `if`;
- a loop body;
- the statements of a `switch` case;
- a `catch` handler;
- the consequent and alternate of a ternary;
- the body of a function passed as an argument.

**Chains.** In `if (a) … else if (b) … else …`, every consequent and the final alternate are at the same depth: a chain is one decision among alternatives, not nested decisions.

**Exiting guards.** An `if` without `else` whose consequent is only `return`, `throw`, `continue`, or `break` (optionally wrapped in a block) opens no context. Its condition and the exit's argument are still measured at the depth where the guard stands.

**Decisions.** `if`, loops, `switch`, `catch` clauses, and ternaries are decisions. A decision's depth is the number of contexts open where it is made, plus one. Logical groups (`&&`, `||`), nullish defaulting, and optional access are not decisions; `max-decision-depth` measures logical nesting inside expressions.

**Only decisions report.** Contexts without a decision inside them are never reported: nested callbacks that decide nothing, such as transaction wrappers and test `describe` bodies, reach no depth.

## Reported code

With `{ "max": 3 }`:

```ts
function carried(items) {
  for (const item of items) {
    if (item.length > 0) {
      item.map((value) => {
        if (value < 0) throw new Error("negative");
        return value;
      });
    }
  }
}
```

The guard is made inside the loop body, the branch, and the callback: depth 4.

```ts
if (a) {
  if (b) {
    if (c) {
      if (d) work();
    }
  }
}
```

Reports `if (d)` at depth 4.

Reports `tooDeep` with the measured depth and configured maximum. The span is the head of the decision: from its keyword to the end of its condition (`if (d)`, `for (const x of xs)`, `switch (kind)`, `catch (error)`), or the condition of a ternary. Only the first over-limit decision in each open context is reported: its siblings in that context and the decisions nested inside it are suppressed, because the burden is the context rather than each decision in it. Separate contexts, including the two branches of one `if`, report separately.

## Accepted code

With `{ "max": 3 }`:

```ts
function validate(a, b, c) {
  if (!a) throw new Error("a");
  if (!b) throw new Error("b");
  if (!c) throw new Error("c");
  return build(a, b, c);
}

function label(kind) {
  if (kind === "a") return "A";
  else if (kind === "b") return "B";
  else if (kind === "c") return "C";
  return "other";
}

function save(items) {
  return perform(async () => {
    await Promise.all(items.map(async (item) => store.save(item)));
  });
}

function factory(a, b, c) {
  return define({
    update() {
      if (a) {
        if (b) {
          if (c) work();
        }
      }
    },
  });
}
```

Guards open no context; the chain stays at depth 1; the callbacks decide nothing; the object method is measured on its own and reaches depth 3.

## Fixes

Diagnostic-only. No fixes or suggestions are emitted. Settling conditions earlier, or giving nested behavior its own owner, can change evaluation order, early exits, and captured state; the rule cannot choose a meaningful name or owner.

## Limitations

- Syntax only. Callbacks passed by identifier, or functions invoked later through an alias, are not followed.
- Whether a callback is invoked immediately, repeatedly, or later is not known; every argument function counts as one context.
- Breadth is deliberately invisible: a flat function with many independent one-level decisions is accepted, although its path count may be a testing burden.
- Interacting state (values whose later use depends on earlier branches) is not measured; that is a separate concern.
- Behavior held in object methods or local closures is measured in those functions, not charged to the enclosing one. How much behavior a function keeps in closures with captured state is a separate ownership concern.
- Test structure counts: `describe`, `it`, and `test` bodies are argument callbacks, so a loop or branch inside a test starts two or three levels deep. The rule does not recognize test files or test functions by name; configure it separately for test globs with Oxlint `overrides` (disable it, or raise `max`).
- Asynchronous plumbing counts too: a promise handler inside a timer callback inside an effect callback is three contexts, even when the decision made there is a one-line cancellation check such as `if (active)`.
