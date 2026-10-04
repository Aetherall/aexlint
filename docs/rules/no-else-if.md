# no-else-if

## Intent

Disallow `else if`. A branch reached through `else if` applies only when its own condition holds and every earlier condition in the chain is false. Its meaning therefore depends on the conditions above it: a reader has to carry the negation of each earlier condition to know when it runs, and reordering, adding, or removing a branch changes the meaning of the branches below it.

This is an opt-in style rule. It conflicts with `max-held-context`, whose contract accepts uniform `else if` chains as already clear; enable one view or the other.

Oxlint 1.83.0 has no rule that reports every `else if`. `no-else-return` with `allowElseIf: false` reports an `else if` only after a branch that returns. `no-lonely-if` and `unicorn/no-lonely-if` report the opposite shape, an `if` alone inside an `else` block, and prefer `else if`. Oxlint does not implement `no-restricted-syntax`.

## Runtime

Oxlint JavaScript plugin; syntax only. No TypeScript type information.

## Reported code

```ts
if (plan === "free") {
  limit = 1;
} else if (plan === "team") {
  limit = 10;
} else {
  limit = 100;
}

function label(user: User) {
  if (user.isAdmin) return "admin";
  else if (user.isOwner) return "owner";
  return "member";
}
```

`elseIf` is reported once per `else if`, on the `else` and `if` keywords. A chain with three `else if` branches has three reports.

```text
This `else if` applies only when every earlier condition in its chain is false, so its meaning depends on the conditions above it.
```

## Accepted code

```ts
if (plan === "free") {
  limit = 1;
} else {
  limit = 100;
}

if (user.isAdmin) return "admin";
if (user.isOwner) return "owner";
return "member";

if (a) {
  run();
} else {
  if (b) log();
  run();
}

const label = isAdmin ? "admin" : isOwner ? "owner" : "member";
```

An `else` block that contains an `if` among other statements is a separate branch, not an `else if`. Conditional expressions are not statements and are not checked.

## Options

None.

## Fixes

Diagnostic-only. Early returns, a lookup table, an exhaustive `switch`, or independent conditions are possible rewrites; none is equivalent in general, so none is applied automatically.

## Limitations

- An `else { if (…) … }` block whose only statement is an `if` behaves like `else if` but is not reported.
- Nested conditional expressions (`a ? x : b ? y : z`) have the same dependency on earlier conditions and are not reported.
