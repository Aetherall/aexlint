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
- A group accepts at least two values and rejects at least two. It is written as comparisons of one subject joined by `||`, `&&` and `!`, `includes` on a literal list or literal tuple, consecutive `case` labels sharing a body, or a literal list whose expected type is a list of the vocabulary.
- Groups match when they accept the same values, whatever the subject, form or order.
- A group that is the whole body of a named function, or a literal list initializing a named variable or property, defines it: definitions are not reported or counted, and help names them.
- The whole program counts, including unlinted files, tests and repeats within one file.

## Reporting and limits

Reports `repeatedValueGroup` on each group in the linted file that also appears elsewhere. Help lists the other places. No fixes.

Unrelated unions sharing values match. Groups built through variables, `switch (true)`, `Set.has`, lookups or helper parameters are not recognized. Other tsconfig programs are not searched.

Serializers legitimately spell out values; turn the rule off for them:

```json
{
  "overrides": [
    { "files": ["**/*.serializer.ts"], "rules": { "aexlint/no-repeated-value-group": "off" } }
  ]
}
```

Overridden files still count when other files are checked.
