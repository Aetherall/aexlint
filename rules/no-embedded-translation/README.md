# no-embedded-translation

Reports branches on a variant that read its fields into the surrounding code's state. Turning a value into your own model is a translation. Written inline, it mixes with the logic that uses it, so that logic reads as mechanics, and a new variant has to be found in every such place. A projection, a function from the value to your model that returns it, keeps the translation in one place. See [projections](../../docs/design/projections.md).

```json
{ "rules": { "aexlint/no-embedded-translation": "warn" } }
```

No options.

## Example

Reported on `instrument.kind`:

```ts
let sources: unknown[];
switch (instrument.kind) {
  case "violin":
    sources = [instrument.bow, instrument.strings];
    break;
  case "oboe":
    sources = [instrument.reed];
    break;
  default:
    return;
}
levels.set(instrument, Math.max(...sources.map(loudness)));
```

Accepted: the translation is a projection, and the logic uses its result.

```ts
function soundOf(instrument: Instrument): Sound | undefined {
  switch (instrument.kind) {
    case "violin":
      return { sources: [instrument.bow, instrument.strings] };
    case "oboe":
      return { sources: [instrument.reed] };
  }
}
```

## What counts

- A decision on a field with finite literal values, as in [no-repeated-value-group](../no-repeated-value-group/README.md#what-counts), whether the type is the project's or a dependency's.
- The tested object's type is a union, and a tested branch reads a field that only some of its members have: content of a variant, not fields every variant shares.
- That branch keeps its result in a local declared outside it, so the surrounding code goes on to use it.
- Branches that return their result, run effects, pass the whole value to a handler, or read only shared fields are accepted. A mode flag on an object that is not a union is not a variant.

## Reporting and limits

Reports `embeddedTranslation` once per decision, on its subject; help lists the values of the embedded branches. No fixes.

A branch whose fields only feed an effect, such as `list.push(x.name)` with nothing kept, is not reported. A returned result that mixes the variant's fields with other state is accepted. A kept local used once afterwards, returned or passed to a constructor, is still reported, although the function is then already a projection.
