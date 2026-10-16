# no-repeated-value-group

Report a group of several values from a literal union that is written out in more than one place. The group stands for a concept the code has no name for, such as "sms or push" meaning "instant channels". When the concept changes, every copy has to be found and changed the same way.

**Status:** experimental. Diagnostic-only; no fixes or suggestions. Usefulness and generality remain unestablished; review findings before adopting it.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/no-repeated-value-group": "warn" }
}
```

**Options:** none. Omitted options, `null`, and `{}` are accepted; anything else is a configuration error. See [typed-plugin setup](../../README.md#type-aware-setup-and-limits). Run `tsc` separately; editor integration is unsupported.

## Example

Reported on both tests, because each spells out the same group:

```ts
type Channel = "sms" | "push" | "email" | "mail";

if (notice.channel === "sms" || notice.channel === "push") sendNow(notice);
// in another file
const urgent = notices.filter((n) => n.uses(["push", "sms"]));
```

Accepted when the group has one definition that both places use:

```ts
const isInstant = (notice: Notice) => notice.channel === "sms" || notice.channel === "push";
const INSTANT: Channel[] = ["sms", "push"];
```

## What counts

- A vocabulary is a union of at least two string or number literals, including enum members, after removing `null` and `undefined`. Generic parameters use their base constraint. Broad types, `any`, `unknown` and unresolved types do not qualify.
- A tested value is compared with a literal, a constant, an enum member, or a property declared with a single literal type, such as a member of an `as const` object.
- The rule uses declared types, not flow-narrowed ones. A property is read from its object's declared type, so narrowing a discriminated union does not change the vocabulary of later tests.
- A group is a test of one subject that accepts at least two values and rejects at least two values. Tests of one value, or of all values but one, do not form a group.
- Groups are written as `===`/`!==`/`==`/`!=` tests combined with `||`, `&&` and `!` on the same subject, as `includes` on a list of literals or a tuple of literal types such as a list written `as const`, as consecutive `case` labels sharing one body, or as a literal list whose expected element type is the vocabulary, such as an argument to a parameter typed `Channel[]`. Other operands may be combined with the group.
- Two groups match when they accept the same values, whatever the vocabulary, subject, form or order. A group and its complement are different groups.
- A test that is the whole body of a named function, method or variable-assigned function, including one passed to a call whose result is callable too, such as `memo(fn)`, and a literal list that initializes a named variable or property, _define_ the group. Definitions are neither reported nor counted; the help names them when they exist.
- The search covers project source files of the TypeScript program, including unlinted files and tests, but not declaration files or files inside `node_modules`. Repetition within one file counts.

## Reporting scope

Reports `repeatedValueGroup` on each group in the linted file that is also written somewhere else. The span is the test, the case labels, or the list. The message names the subject and the group, and does not depend on other files. Help gives the number of places, lists the nearest others first, and names existing definitions.

## Boundaries

Serializers and other translation boundaries legitimately spell out values. Turn the rule off for them with Oxlint overrides:

```json
{
  "overrides": [
    { "files": ["**/*.serializer.ts"], "rules": { "aexlint-typed/no-repeated-value-group": "off" } }
  ]
}
```

Groups inside overridden files still count when other files are checked.

## Limitations

- Identity is the set of values. Unrelated unions that share the same values are matched.
- Groups built through intermediate variables, `switch (true)`, `Set.has`, object lookups or helper parameters are not recognized. Lists without an expected vocabulary type, such as an untyped constant, are not recognized.
- A literal list is matched only when its expected type is a list of the vocabulary; tuple types use their first element type.
- Other tsconfig programs are not searched.
