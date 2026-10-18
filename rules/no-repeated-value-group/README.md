# no-repeated-value-group

Reports a group of values from a literal union written out in more than one place. The group stands for an unnamed concept, such as "sms or push" meaning "instant channels"; when the concept changes, every copy must change the same way.

```json
{ "rules": { "aexlint/no-repeated-value-group": "warn" } }
```

No options.

## Example

Reported on both tests:

```ts
type Channel = "sms" | "push" | "email" | "mail";
if (notice.channel === "sms" || notice.channel === "push") sendNow(notice);
// another file
const urgent = notices.filter((n) => n.uses(["push", "sms"]));
```

Accepted when both places use one definition:

```ts
const isInstant = (notice: Notice) => notice.channel === "sms" || notice.channel === "push";
```

## What counts

- A vocabulary is a union of at least two string or number literals (enum members included), ignoring `null` and `undefined`, read from the declared type. Generic parameters use their constraint; `any`, `unknown` and unresolved types do not qualify.
- A group accepts at least two values and rejects at least two. It is written as comparisons of one subject joined by `||`, `&&` and `!`, `includes` or `has` on a literal list, literal tuple or `new Set` of a literal list, consecutive `case` labels sharing a body, a literal list whose expected type is a list of the vocabulary, or an object literal mapping every value to `true` or `false` where a `Record` of the vocabulary to `boolean` is expected (the values mapped to `true`).
- Groups match when they accept the same values, whatever the subject, form or order.
- A definition names a group: a named function whose whole body it is, a literal list or table initializing a named variable or property, or a `const` list or set it is tested through. Testing through a definition's name uses it rather than copying it.
- A copy matching a definition is reported, and help names the definition. Two definitions of one group are both reported, once each.
- The whole program counts, including unlinted files, tests and repeats within one file.

## Reporting and limits

Reports `repeatedValueGroup` on each group in the linted file that also appears elsewhere. Help lists the other places. No fixes.

Unrelated unions sharing values match. Groups built through other variables, `switch (true)`, partial tables, lookups or helper parameters are not recognized, and a `const` collection changed after it is declared is read as declared. Other tsconfig programs are not searched.

Serializers legitimately spell out values; turn the rule off for them:

```json
{
  "overrides": [
    { "files": ["**/*.serializer.ts"], "rules": { "aexlint/no-repeated-value-group": "off" } }
  ]
}
```

Overridden files still count when other files are checked.
