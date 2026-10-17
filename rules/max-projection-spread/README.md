# max-projection-spread

Limits how many files repeat how a project value becomes an output object. When an output field's meaning changes, each copy needs the same change.

```json
{ "rules": { "aexlint/max-projection-spread": ["warn", { "max": 2 }] } }
```

`max` is an optional positive integer, default 2.

## Example

Reported when three files build:

```ts
return {
  id: site.id.serialize(),
  shining: site.getBrightness().is("shining"),
};
```

Accepted: the projection lives on `Site`, and callers use `site.toResponse()`.

```ts
toResponse() {
  return { id: this.id.serialize(), shining: this.getBrightness().is("shining") };
}
```

## What counts

- A derivation is an object property with a static key, built from a variable through member reads and at least one method call with literal or no arguments. Its type must be declared in project code.
- A projection needs at least two derivations from the same variable.
- Derivations match on output key, owner type (with type arguments) and the chain of members and literal arguments; variable names do not matter. Files are counted per derivation across the whole program.
- Plain copies (`id: site.id`), shorthand, `this` roots, helper calls and non-literal arguments do not count.

## Reporting and limits

Above `max`, reports `spreadProjection` once per projection, on its first over-limit key. Help gives each derivation's file count. No fixes.

Versioned serializers and read models may repeat derivations on purpose; turn the rule off for them with overrides. Whole-object similarity, intermediate variables and other tsconfig programs are not considered.
