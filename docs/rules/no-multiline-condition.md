# no-multiline-condition

Experimental, syntax-only Oxlint JS rule. Keeps statement conditions on one line so the decision can be read before its body.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/no-multiline-condition": "warn" }
}
```

No options, fixes or suggestions.

## Example

Reported:

```ts
if (user.isActive && user.subscription.plan !== "free" && !user.flags.suspended) {
  grantAccess(user);
}
```

Accepted:

```ts
const canAccess = user.isActive && user.subscription.plan !== "free" && !user.flags.suspended;
if (canAccess) {
  grantAccess(user);
}
```

## Reporting and limits

Checks `if`/`else if`, `while`, `do … while` and the test clause of ordinary `for` loops. Reports `multiline` when the condition's first and last characters are on different lines, spanning the condition without the statement's parentheses. The message includes statement kind and line count.

Only the condition is measured. Parentheses on separate lines or a multiline `for` header pass when the test itself is on one line. Every internal line break counts, including those in comments, templates, callbacks and object literals.

Ternaries, switch discriminants, case tests, `for … in`/`for … of` heads and expressions outside statement conditions are not checked.

This measures layout, not complexity. A formatter's wrapping choices affect reports. Naming a decision is one possible response, but moving expressions can change evaluation order or short-circuiting, especially in loop tests; no automatic rewrite is offered.
