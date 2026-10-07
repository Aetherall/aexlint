# max-call-assembly

Experimental, syntax/scope-only Oxlint JS rule. Reports calls whose arguments are too wide to read on one line and also draw on many names from the surrounding code.

A call that cannot be read as one line and pulls many separate values from its surroundings makes the reader recall what each of them is and work out how they fit together at that one point. Width alone is not the problem: declarative data such as rule metadata, test tables and long strings can be wide while referring to almost nothing. A handful of names alone is not the problem either: `read(fd, buffer, offset, size)` is short. The rule reports only when both are present.

## Configuration

```json
{
  "jsPlugins": ["@aetherall/aexlint"],
  "rules": { "aexlint/max-call-assembly": ["warn", { "maxWidth": 80, "maxReferences": 3 }] }
}
```

Exactly one options object is required, with both `maxWidth` and `maxReferences` as non-negative safe integers; unknown keys and extra options are rejected. A call is reported when its width exceeds `maxWidth` and its outside names exceed `maxReferences`. There is no default or recommended limit.

## What counts

Only calls inside a function or class body are checked. Code at module level, such as schema, event, style and plugin declarations, runs once to declare structure; code inside functions assembles values each time it runs.

React hooks are not checked: calls to `use` or to a name starting with `use` followed by an uppercase letter or digit, directly or as a member such as `React.useMemo`. This exclusion is provisional and not settled. Hook calls are often wide because of their dependency arrays, which React requires; whether that width is assembly worth reporting is undecided. It is also a naming-based exception, unlike the rest of the rule. Calls inside hook callbacks are still checked.

The argument list of each checked call and `new` expression, from `(` to `)` inclusive, is measured. Calls without arguments are ignored. The callee, type arguments and `?.` are not measured, so long method chains and receivers do not count.

Width is the argument list as if printed on one line:

- Function and class bodies anywhere in the arguments count as `{}`, whether the function is a direct argument, an object property or nested deeper. Parameters, defaults and type annotations still count.
- Each whitespace or comment gap between tokens counts as one space, except directly after `(` or `[` and directly before `)` or `]`. Comments themselves do not count.
- Trailing commas before `)`, `]` or `}` do not count.
- Literal text does not count: a string counts as its two quotes, a template keeps only its delimiters and interpolations (`` `a ${b} c` `` counts as `` `${b}` ``), and JSX text counts as nothing. Interpolated expressions count in full.

Outside names are the distinct value references in the arguments, outside function and class bodies, whose declaration is not inside the argument list. Locals, imports, globals and unresolved names all count; parameters of a callback passed in the arguments do not. A member path counts its root once: `config.port` and `config.host` contribute `config`. A callee inside the arguments counts, as do JSX component and attribute value names. Names in type annotations, type arguments and `as`/`satisfies` types do not count.

## Example

With `{ "maxWidth": 60, "maxReferences": 3 }`, this reports the `createServer` call: width 75, with 5 outside names (`config`, `resolveHost`, `env`, `loadCerts`, `paths`):

```ts
function start(config: Config, env: Env, paths: Paths) {
  return createServer({ port: config.port, host: resolveHost(env), tls: loadCerts(paths.cert) });
}
```

Accepted with the same limits:

```ts
items.map((item) => {
  const normalized = normalize(item);
  return encode(normalized, options);
});

on("click", { once: true, handler: () => close(dialog) });

export const styles = StyleSheet.create({
  box: { color: Color.primary, margin: Spacing.small, radius: Radius.medium },
});

const visible = useMemo(() => filter(items), [items, query, sortOrder, filters, locale]);
```

## Reporting and limits

Reports `tooAssembled` from the opening to the closing parenthesis, with the width, the number and names of outside names, and both limits. When an over-limit call sits in the measured arguments of another call, only the inner call is reported; calls inside function bodies are measured independently.

No fixes or suggestions. Naming arguments first can change evaluation order or scope.

Both measures are proxies. Declarative tables whose many entries repeat the same few names, such as test tables using several helpers, can exceed both limits. Long names make a call wide without making it harder to follow. Utility names such as `join` or `JSON` count like any other reference. Shadowed names that share text count once. Module-level scripts that do their work outside functions are not checked. Tagged templates and dynamic `import()` are not checked. Widths are UTF-16 code units.
