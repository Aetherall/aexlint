# no-split-result

Reports a function result whose optional fields its returns always set together, or never together. The result type allows combinations no return builds, so each reader tests the fields one by one and decides what a partial combination means. The pairing the function knows is restated, or guessed, at every use.

```json
{ "rules": { "aexlint/no-split-result": "warn" } }
```

No options.

## Example

Reported on `: Digest`: `summary` and `tokens` come together, and never with `warning`.

```ts
interface Digest {
  summary?: string;
  warning?: string;
  tokens?: number;
}
function digest(mode: Mode, text: string): Digest {
  if (mode === "off") return {};
  if (mode === "local") return { summary: text, tokens: text.length };
  return { warning: "remote digest unavailable" };
}
```

Reported on `: Link`: `label` and `url` are set together or not at all.

```ts
const link = (path: string | undefined, internal: boolean): Link => {
  if (!path) return {};
  if (internal) return { label: path, url: `/pipelines/${path}` };
  return { label: path, url: `https://ci.example/${path}` };
};
```

Accepted: `{ summary: string; tokens: number } | { warning: string }` as the result, an `undefined` result instead of `{}`, or returns that build several combinations of the same fields (`{ width }`, `{ height }`, `{ width, height }`).

Only the result type decides: with `type Reading = { celsius: number } | { fault: string }` imported from another file nothing is reported; changing that file to `{ celsius?: number; fault?: string }` reports the function.

## Fixing it

Declare the combinations the returns already build. The function usually compiles unchanged; only the type and its readers change.

```ts
interface Upload {
  state: "queued" | "sending" | "done" | "failed";
  sent?: number;
  error?: string;
}
function next(event: UploadEvent): Upload {
  switch (event.type) {
    case "queued":
      return { state: "queued" };
    case "progress":
      return { state: "sending", sent: event.sent };
    case "finished":
      return { state: "done", sent: event.sent };
    case "failed":
      return { state: "failed", error: event.message };
  }
}
function label(upload: Upload) {
  if (upload.state === "failed") return `Failed: ${upload.error ?? "unknown error"}`;
  return `Sent ${upload.sent ?? 0} bytes`;
}
```

becomes:

```ts
type Upload =
  | { state: "queued" }
  | { state: "sending" | "done"; sent: number }
  | { state: "failed"; error: string };
function label(upload: Upload) {
  if (upload.state === "failed") return `Failed: ${upload.error}`;
  if (upload.state === "queued") return "Queued";
  return `Sent ${upload.sent} bytes`;
}
```

`next` is unchanged. The fallbacks `?? "unknown error"` and `?? 0` were answers for states no code builds: with the optional shape they compile, and when a bug does produce such a state they show wrong data instead of failing. The union also made `label` handle a queued upload, which it showed as "Sent 0 bytes": `sent` does not exist there, so the old last line no longer compiles.

Other shapes:

- Fields set together or not at all: group them in one optional value, `{ link?: { label: string; url: string } }`, or return `undefined` instead of `{}`.
- The shape depends on which branch of a `switch (this.kind)` built it, and later methods switch on `this.kind` again with `!` on the fields: put the tag in the result (`{ position: "refining"; subject; prefix }`) and switch on that, or give each kind its own object.

"The optional shape is simpler" holds for the declaration only. The cost moves to every reader: each one tests the fields one by one, adds a fallback or a `!` for combinations that cannot occur, and must be found by hand when a state or a field is added, since the type says every combination is valid.

## What counts

- A function with a body whose result, awaited and without `null` and `undefined`, is one object type declared in the project: not a union, an intersection, a callable type, `any`, `unknown` or an unresolved type. A result with literal types per return, as TypeScript infers for several object literals, is a union and is not read.
- Every return gives an object literal, a conditional between them, or `null`/`undefined`, with at least three object literals. With two, any fields that differ look paired by chance.
- A field is optional when it is marked `?` or its type includes `undefined`. For each optional field, a literal sets it, leaves it out (missing or `undefined`), or may set it: a value whose type includes `undefined`, `any` or `unknown`, or a spread whose type has the field. A field some literal may set is not read.
- Fields set by some literals and left out by the others are grouped by the literals that set them. A group of two or more fields is set together or not at all; two groups no literal sets together are never set together.

## Reporting and limits

Reports `splitResult` on the written result type, otherwise on the function, once per function, listing every pairing. No fixes: the union's members may need names.

Results built through helper calls, local variables or a call's result, and correlations between fields of a type used by several functions, are not read. Readers of the result are not inspected: the rule reports what the type allows, not a reader that mishandles it.
