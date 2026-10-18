# max-call-assembly

Reports calls whose arguments are too wide to read on one line and also draw on many names from the surrounding code. The reader has to recall each of those values and work out how they fit together at one point. Width alone (declarative tables, long strings) or a few names alone (`read(fd, buffer, offset, size)`) is not reported.

```json
{ "rules": { "aexlint/max-call-assembly": ["warn", { "maxWidth": 60, "maxReferences": 3 }] } }
```

Both options are required non-negative integers. A call is reported when its width exceeds `maxWidth` and its outside names exceed `maxReferences`.

## Example

Reported (width 75, outside names `config`, `resolveHost`, `env`, `loadCerts`, `paths`):

```ts
function start(config: Config, env: Env, paths: Paths) {
  return createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });
}
```

Accepted:

```ts
on("click", { once: true, handler: () => close(dialog) });
export const styles = StyleSheet.create({ box: { color: Color.primary, margin: Spacing.small } });
```

## What counts

- Only calls and `new` inside function or class bodies; module-level declarations are skipped. Calls without arguments are ignored.
- Width is the argument list printed on one line: nested function and class bodies count as `{}`, gaps as one space, comments and trailing commas as nothing, and literal text as nothing (a string counts as its quotes).
- Outside names are distinct value references in the arguments declared outside them. A member path counts its root once, and `this` counts as one name; callback parameters, `this` inside nested `function` bodies, and type positions do not count.
- An argument, or spread argument, naming a `const` object or array literal that is read only there, by a statement of the same list, is measured as that literal: building the arguments one statement earlier assembles the same names. A literal read elsewhere too, built by a call, or declared with `let` is a value of its own.
- React hook calls (`use`, `useX`, `React.useX`) are skipped because of their dependency arrays. This is a provisional, name-based exception.

## Reporting and limits

Reports `tooAssembled` on the argument list, with width, names and limits. Only the innermost over-limit call in nested arguments is reported. No fixes.

Both measures are proxies: long names widen calls, test tables repeating a few helpers can exceed both limits, and shadowed names sharing text count once. Tagged templates and `import()` are not checked. Literals are followed one step, not through other locals or helpers, and a single-use literal with a well-chosen name is measured like any other.
