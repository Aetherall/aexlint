# max-expression-depth

Limits how deeply calls are nested inside other calls' inputs.

```json
{ "rules": { "aexlint/max-expression-depth": ["warn", { "max": 2 }] } }
```

`max` is a required positive integer.

## Example

Reported at `max: 2` (depth 3):

```ts
publish(encode(parse(input)));
```

Accepted:

```ts
const document = parse(input);
publish(encode(document));
```

## What counts

- A call, construction, tagged template or `import()` is one deeper than its deepest input. `factory()()` has depth 2.
- A method chain is a sequence, not nesting: `source.first().second()` has depth 1. Its arguments still count.
- Other syntax passes the deepest child through. Function and class bodies are checked on their own.

## Reporting and limits

Reports `tooDeep` on the outermost over-limit call; nested reports are suppressed. No fixes: extracting inputs can change evaluation order. Getters are not calls. Other expression rules may report the same expression.
