# no-else-if

Reports every `else if`: reaching that branch depends on its own condition and on every earlier one being false.

```json
{ "rules": { "aexlint/no-else-if": "warn" } }
```

No options.

## Example

Reported:

```ts
if (day.isStormy) return "stormy";
else if (day.isCloudy) return "cloudy";
return "sunny";
```

Accepted:

```ts
if (day.isStormy) return "stormy";
if (day.isCloudy) return "cloudy";
return "sunny";
```

## Reporting and limits

Reports `elseIf` once per `else if`, from `else` through `if`. Plain `else`, ternaries and `else { if … }` are not checked. Wrapping the `if` in a block is not a fix: the reader still holds every earlier condition, and Oxlint's `no-lonely-if` rewrites a lone `if` in an `else` block back into `else if`. Use early returns. No autofix: early returns, switches and lookup tables are not interchangeable in general.

[max-held-context](../max-held-context/README.md) allows `else if` chains. The early returns above satisfy both rules.
