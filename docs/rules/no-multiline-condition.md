# no-multiline-condition

Keeps statement conditions on one line so the decision can be read before its body.

```json
{ "rules": { "aexlint/no-multiline-condition": "warn" } }
```

No options.

## Example

Reported:

```ts
if (
  user.isActive &&
  user.subscription.plan !== "free" &&
  !user.flags.suspended &&
  !user.flags.locked
) {
  grantAccess(user);
}
```

Accepted:

```ts
const canAccess =
  user.isActive && user.subscription.plan !== "free" && !user.flags.suspended && !user.flags.locked;
if (canAccess) grantAccess(user);
```

## Reporting and limits

Checks `if`, `while`, `do … while` and the test of `for`. Reports `multiline` on a condition whose first and last characters are on different lines, excluding the statement's parentheses. Every line break counts, including in comments, templates and callbacks. Ternaries, `switch` and `for … of`/`in` heads are not checked.

This measures layout, so formatter wrapping affects it. No fixes: moving a loop test can change evaluation order.
