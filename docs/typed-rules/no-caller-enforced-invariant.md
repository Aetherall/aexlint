# no-caller-enforced-invariant

## Intent

Report a check that a caller makes on data before handing it to a project class that stores the data without that check. The rule the check enforces then lives with the caller rather than with the data:

- other code that creates or changes the class's state is not held to the rule, and nothing points to it;
- each caller that wants the rule must repeat the check, and the copies can drift apart.

The rule reports this observation, not a diagnosis. The same observation comes from several design problems, and sometimes from none:

- the class, or a value type it takes, should own the rule (`Duration.fromSeconds` accepting `NaN` after its caller checked for it);
- the class's parameters carry information the caller must keep consistent (a site's identifier and its team passed separately, after checking that they agree);
- a domain operation is missing, so its rules live in the handler (a "cannot delete this site" check followed by building the deleted record directly);
- or the rule only applies to the calling operation and belongs where it is (an upload size limit of one endpoint, a feature that must be installed first).

```ts
// credential.ts
export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential {
    return new Credential(expiresAt);
  }
}

// issue-credential.ts
const expiresAt = new Date(command.expiresAt);
if (Number.isNaN(expiresAt.getTime()) || expiresAt <= new Date())
  throw new Error("expiresAt must be in the future");
return Credential.create(expiresAt);
```

The diagnostic states the separation and the possible causes. It does not prescribe a remedy.

Status: experimental research candidate. Not ready for promotion: see [Evaluation](#evaluation).

`no-forgeable-validated-result` reports a check whose result is returned as a plain value. This rule reports a check whose data is handed to a class that does not own it. They share their data-flow analysis; see that rule's [Measurement](no-forgeable-validated-result.md#measurement) for paths, input, narrowing tests, and calls.

## Runtime and options

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/no-caller-enforced-invariant": "warn"
  }
}
```

No options. Options other than an empty object produce a backend configuration error.

## Measurement

**Callers.** Every function, method, constructor, and function expression with a body in the linted file.

**Checks.** A rejection of input in the caller's body, outside nested functions and classes, as defined by `no-forgeable-validated-result`: a `throw` under a condition that reads input and is not a narrowing test, or a call to a project function that rejects input it is passed. A check reads all the input data its conditions read, including data passed into calls whose result they test: `isNaN(expiresAt.getTime())` reads `expiresAt`.

**Data.** Data is compared as access paths, as in the shared analysis, and an overlap must be definite: two unknown parts of one value, such as the results of two getters on it (`user.getFirstName()`, `user.getLastName()`), are not the same data. Data read through `this` (injected services, the caller's own fields) is recorded as context: it never makes data input, but a check that reads it reads data a class it hands values to is not given.

**Order.** The check must be able to run before the call on the way to it: the statement holding it comes before the call in a statement list enclosing both, and no branch between the check and that statement returns or otherwise leaves before the call. A check in another branch of the same `if`, or in a branch that returns first, does not precede the call. A check nested in conditions, applied only for some actors or values, still precedes a call after them: the observation holds for the values it applies to.

**Hand-offs.** A call or `new` expression after the check, in the same body, whose resolved declaration is a constructor or method of a class declared in a project source file, with a body. A construction or static factory whose result is discarded is not a hand-off: nothing receives the instance, as when a factory is called only to see whether it throws. Calls from inside that class or a class deriving from it, and calls on `this` or `super`, are not hand-offs: the check is already within the owner's code, including when the class extends a mixin whose members the checker cannot attribute to a class declaration.

**Stored parameters.** The callee takes a parameter into its class's state when it is a constructor parameter property; is assigned to a property of `this`; is passed to a method of a property of `this` whose result is discarded (`this.events.push(value);`); is passed to a member called on `this` or `super`, or to a constructor or member of the same class, that stores it; or, for a static method returning an instance of its class (directly or as a property of the returned object), is read at all. Queries such as `hasMember(name)` and loggers take nothing. The same notion decides which arguments a method call on a local stores into that local in the shared analysis.

**Reported argument.** An argument of a hand-off at a parameter the callee stores but does not check, that contains data a preceding check read, unchanged: the checked value itself, or a value holding it (`{ site }`, `site.id`), not a value computed from it (`name.split(":")[1]`), selected by it (`format === "a" ? a : b`), or derived through a method call. The overlap must be definite. The callee checks a parameter when its own conditions, the project functions it calls, or the value types it constructs (`FirstName.from(first)`, throwing on an empty name) reject input read from it. A check made by constructing such a value type in the caller is that type's own check, not a check before a hand-off.

**Coverage.** The message states whether the check reads only data handed to the class at stored parameters, in which case the class could make the check itself, or names the first datum it also reads that the class does not receive, such as the acting user or an injected service. A method call in a check reads an unknown part of its receiver as well as its arguments, since the method may check its receiver's state: `actor.ensure(teamId)` reads part of `actor`. When several checks precede an argument, the first one in source order whose data overlaps it is reported.

## Reported code

The example above. `checkedBeforeHandOff` is reported on the argument (`expiresAt`), once per argument:

```text
`expiresAt` is checked at line 3, then handed to `Credential.create`, which stores it without that check.
help: The rule this check enforces lives with this caller rather than with the data, so other code that creates or changes a `Credential` is not held to it. The check reads only data `Credential.create` receives. This often means `Credential`, or a value type it takes, should own the rule; that its parameters carry information the caller must keep consistent; or that a domain operation is missing. If the rule only applies to this operation, it belongs here.
```

A check that also reads context names it:

```ts
if (!(await this.subscriptions.enabled(command.teamId))) throw new Error("not installed");
return Member.enlist(command.teamId);
```

```text
`command.teamId` is checked at line 1, then handed to `Member.enlist`, which stores it without that check.
help: … The check also reads part of `this.subscriptions`, which `Member.enlist` does not receive. …
```

## Accepted code

```ts
export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential {
    if (Number.isNaN(expiresAt.getTime()) || expiresAt <= new Date())
      throw new Error("expiresAt must be in the future");
    return new Credential(expiresAt);
  }
}

const credential = Credential.create(new Date(command.expiresAt));
```

Also accepted:

- a check of data that is not passed to the class, or that is passed after being replaced;
- a check made after the call;
- a narrowing test only (`if (!value) throw`, `if (!(value instanceof X)) throw`, an assertion helper declared `asserts value`);
- a check in another branch of the same `if`, or in a branch that returns before the call;
- a value computed from or selected by the checked data;
- a factory called only to see whether it throws;
- a hand-off to a plain function, to a library class, or to a member of the same class or a base class;
- a callee that checks the parameter itself, directly, through a project function it calls, or through a value type it constructs;
- data sharing only a parent object with the checked data (`if (first.length > 20) throw …; Person.named(first, user.getLast())`).

Cross-file example: the linted file checks `expiresAt` and calls `Credential.create(expiresAt)` from another file. When `create` does not check its parameter, the argument is reported; changing only the other file so `create` throws on a past date makes the linted file accepted.

## Evaluation

First labeled evaluation, 2026-10-04, of the 32 reports whose check read only data handed to the class, on an application backend of about 4,600 files (58 reports whose check also read other data were not labeled), before the data and ownership refinements under [Measurement](#measurement). Corpus names and per-report labels are kept outside the repository.

| Label          | Reports | Patterns                                                                                                                                                                                                                                                                                                                                                                                                |
| -------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Useful         | 3       | A validity check on a date copied into two handlers before a factory that stores any date; a finiteness check before a value type that declares but never runs its own validity rule                                                                                                                                                                                                                    |
| Debatable      | 7       | A redundant parameter checked by one caller and hard-coded by others; an upload policy repeated at some entry points and missing at others, whose owner would be a dedicated value type; a "required field" rule already owned by the caller's own factory                                                                                                                                              |
| False positive | 22      | Data sharing a root with the checked data but not checked itself (6); preconditions reading injected stores or services, which count as no data (4); authorization and authentication (2); callees that delegate the check to a value type they construct (2); migration scripts (2); checks owned by the caller's own class over its collection (3); boundary policy (1); event types (1); lookups (1) |

Precision was about 9% counting only useful reports, against the narrower question "should the class own this rule?". The useful findings are the ones where several callers repeat the check or the class declares the rule without enforcing it.

After the refinements (definite overlap, context read through `this`, checks made through value types, subclasses as owners), the same backend has 74 reports. Of the 32 labeled ones, all 3 useful reports remain; 9 false positives and 1 debatable report are gone. Most remaining false positives are rules legitimately owned by the operation (installation and extension preconditions, an endpoint's upload limit, uniqueness enforced by the aggregate that creates the child), which the message now allows the reader to dismiss by naming the context the check also reads. Relabeling of all 73 reports against the question the message asks, whether knowledge about the data is misplaced for any reason:

| Category                                               | Reads only handed data | Also reads other data | Total |
| ------------------------------------------------------ | ---------------------- | --------------------- | ----- |
| The class or a value type it takes should own the rule | 3                      | 2                     | 5     |
| Redundant parameters the caller keeps consistent       | 0                      | 3                     | 3     |
| Missing domain operation                               | 1                      | 1                     | 2     |
| Missing value type                                     | 1                      | 1                     | 2     |
| Rule legitimately owned by the operation               | 9                      | 41                    | 50    |
| Observation false                                      | 2                      | 9                     | 11    |

12 reports (16%) point at misplaced knowledge; 31% of those whose check reads only handed data, 12% of the others. About 38 of the legitimately placed rules are authorization checks on the acting user, repeated across handlers. The coverage fact pointed the reader toward the right category in 48 of 73 reports. The false observations came from a check in a branch that does not run on every path to the call (5), an argument computed from or selected by the checked data (2), an assertion helper not recognized as a presence test (1), and a factory called only to see whether it throws (1); two more were correct observations of legitimately placed rules (authorization over a path containing the checked identifier, and wiring in a composition root). After the order, unchanged-data, assertion, and discarded-result refinements, the 12 misplaced-knowledge findings and all 50 legitimately placed rules remain; 7 of the 9 false observations are gone, and the 2 that remain are checks in one branch preceding a call in a separate, mutually exclusive branch.

## Fixes

Diagnostic-only.

## Limitations

- **Unknown parts are not covered.** A check reading an unknown part of a value (`isValid(input)`), followed by handing over specific fields (`input.name`), is described as also reading other data, since the analysis cannot tell which part the check read.
- **Every check before the call counts.** A check of data that reaches the argument is treated as a requirement of the callee, although it may be a requirement of the caller's own operation that happens to read the same data.
- **Only classes are owners.** A function-based module that owns its data through factories is not recognized as an owner.
- **Mutually exclusive branches.** A check in one conditional branch precedes a call in a later, separate conditional branch, even when the two conditions cannot both hold (`instanceof A` and then `instanceof B` for unrelated classes): telling those apart needs reasoning about values the analysis does not do.
- **Same limits as the shared analysis.** Data flow through callbacks, reassigned variables, and rest parameters is not followed; `this` state is context, not input.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`. Editor integration is not supported yet.
