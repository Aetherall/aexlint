# no-multiline-condition

## Intent

Keep the condition of a control-flow statement on one line. The condition decides whether, or how often, the body runs. When it spans several lines, a reader has to assemble the whole decision before they can tell when the branch applies, and the statement's head no longer reads as one decision. This often happens without anyone choosing it: the formatter wraps a condition that has grown past the print width.

Oxlint 1.83.0 has no stylistic line-break rules. `no-unexpected-multiline` only reports line breaks that change how automatic semicolon insertion parses the code, and `max-len` is not implemented. Formatters such as Oxfmt decide where conditions wrap but never refuse to wrap them.

## Runtime

Oxlint JavaScript plugin; syntax only. No TypeScript type information.

## Reported code

The rule checks the condition of:

- `if` and `else if` statements;
- `while` and `do … while` loops;
- the test clause of a `for (init; test; update)` loop.

It reports when the condition's first and last characters are on different lines. The diagnostic spans the condition itself, without the statement's parentheses.

```ts
if (
  user.isActive &&
  user.subscription.plan !== "free" &&
  user.subscription.renewsAt > now &&
  !user.flags.suspended
) {
  grantAccess(user);
}

if (
  items.some((item) => {
    return item.expired;
  })
) {
  purge(items);
}
```

Both are reported. The second condition is a single call, but the callback body is part of the condition, so the condition still spans several lines. The same applies to `while`, `do … while` and `for` tests.

## Accepted code

```ts
const canAccess =
  user.isActive &&
  user.subscription.plan !== "free" &&
  user.subscription.renewsAt > now &&
  !user.flags.suspended;
if (canAccess) {
  grantAccess(user);
}

if (items.some(isExpired)) {
  purge(items);
}
```

- Only the condition is measured, not the statement's parentheses, braces or body. A `for` header that spans several lines is accepted while its test clause is on one line.
- A multi-line expression elsewhere, such as the `const` above or a ternary, is not a control-flow statement condition. Naming the decision is one way to keep the condition on one line; a shorter expression or a different split of the decision are others.

## Options

None.

## Fixes

None. A one-line condition usually needs part of the decision moved somewhere else, and the rule cannot choose a meaningful name or prove that moving an expression preserves evaluation order and short-circuiting.

## Limitations

- Ternaries, `switch` discriminants, `case` tests, `for … in`/`for … of` heads and logical expressions used as statements are not checked. They are not statement conditions.
- Every line break inside the condition counts, including one introduced by a comment, a template literal, a multi-line callback or an object literal.
- With a formatter, the rule effectively reports conditions that do not fit the configured print width, plus conditions the formatter breaks for other reasons such as a multi-line callback. It does not measure how complex a condition is; `max-expression-complexity` and `max-decision-depth` do.
