# no-repeated-conversion

Reports a conversion of a variant value written identically in several places. When the conversion changes, every copy must change the same way.

```json
{ "rules": { "aexlint/no-repeated-conversion": "warn" } }
```

No options.

## Example

Reported in both files:

```ts
// toolbar.tsx
item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };
// dropZone.tsx
dragged.target === "cell" ? { type: "cell", template: dragged.template } : { type: "row" };
```

Accepted: a conversion written once, or conversions that differ in any key or value.

## What counts

- A dispatch is a `switch`, `if`/`else` chain or ternary chain on a finite literal property, as in [no-external-variant-behavior](../no-external-variant-behavior/README.md#what-counts). Its constructions are the object and array literals each branch builds from the tested value, with where they go.
- Two dispatches are copies when they test the same property (same name and a shared declaration), cover the same values with complete conditions, and build identical constructions for each value.
- Constructions are compared relative to the tested value, so its name does not matter. Literals compare by value; external references and callees must resolve to the same symbol. Object keys may be reordered only when their values are plain reads or literals.
- Copies may be in the same file or anywhere in the program.

## Reporting and limits

Reports `repeatedConversion` on the tested property of each copied dispatch; help lists the other copies. No fixes.

Exact matches only: a renamed local or extra field breaks the match, so drifted copies are missed. Array-only constructions are not compared. Types declared twice are different subjects. Other tsconfig programs are not searched. Versioned serializers repeat conversions on purpose; turn the rule off for them with [overrides](../no-repeated-value-group/README.md#reporting-and-limits).
