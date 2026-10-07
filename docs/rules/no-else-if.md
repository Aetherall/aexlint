# no-else-if

Experimental, syntax-only Oxlint JS style rule. Reports every `else if`: reaching that branch depends on its condition being true and every earlier condition being false.

This conflicts with [max-held-context](max-held-context.md), which treats uniform `else if` chains as clear alternatives. Choose the view that suits your code.

## Configuration

```json
{
  "jsPlugins": ["aexlint"],
  "rules": { "aexlint/no-else-if": "warn" }
}
```

No options, fixes or suggestions.

## Example

Reported:

```ts
function label(user: User) {
  if (user.isAdmin) return "admin";
  else if (user.isOwner) return "owner";
  return "member";
}
```

Accepted:

```ts
function label(user: User) {
  if (user.isAdmin) return "admin";
  if (user.isOwner) return "owner";
  return "member";
}
```

## Reporting and limits

Reports `elseIf` once per `else if`, from the start of `else` through the end of `if`, including intervening whitespace or comments. A chain with three `else if` branches produces three reports.

Plain `else`, nested ternaries and `else { if (…) … }` are not checked, even when the block contains only that `if`. The rule checks statement shape, not equivalent behavior.

Early returns, lookup tables, switches and independent conditions are not interchangeable in general. No rewrite is applied automatically.
