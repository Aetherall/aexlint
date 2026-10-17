# Decision model

`native/model/` reads what code knows about values, once, from types and syntax. Typed rules are queries over it. Code knows **facts**: a test establishes one, it holds for a while, and code uses it. The rules find facts that are lost and rebuilt, established twice, established outside their owner, or spread across files.

## Glossary

- **Value**: data reached from a root (a variable, `this`, or an expression evaluated once) through property reads and literal-key element reads. Identity comes from symbols, not text; two calls written alike are different values. A variable that only names data is that data: after `const { kind } = item`, `kind` is `item.kind`.
- **Field**: the property declarations a value ends at. Two values are the same field when they share a name and a declaration.
- **Form**: a structural comparison key for an expression relative to a value: the value becomes `$`, literals compare by value, and external references by resolved symbol.
- **Vocabulary**: the finite literal values of a declared type, at least two string or number literals once `null` and `undefined` are removed.
- **Claim**: what a test establishes: the value is one of a set, within its vocabulary or among present, `undefined`, `null` and other falsy values. Claims compare as sets: same, opposite, or implies.
- **Test**: a place where code learns a claim: a literal comparison, `includes` on a list of known values, or a presence test (a truthiness test or a comparison with `null`/`undefined`).
- **Rejection**: a place that rejects input without the model reading its claim: a `throw` under conditions reading input, or a call to a function that makes one. Narrowing-only tests are not rejections. In rule messages this is a "check".
- **Guard** and **entry test**: a guard is a test after which code works only where the claim holds and otherwise leaves without using the value. An entry test is a function's guard on a parameter before its work; its **fallback** is the constant it leaves with.
- **Leaves**: returns, throws, breaks, continues, calls a function returning `never`, or ends in branches that all leave.
- **Boundary**: where a value passes between functions: an argument to a parameter, a call's result, a store into a class's state, or a **gate** (a selector and handler in one object or class taking the same first parameter type). The questions there are what is known on the way in, whether the receiving type keeps it, and whether the receiver rejects it again.
- **Flow**: what a function does with its input: the data each value is built from, the rejections it makes, the calls it passes data to, and what it returns.
- **Owner**: who a type belongs to: its class, a runtime value merged with it, or its declaring module or namespace. Importing a type does not transfer ownership.
- **Predicate**: a parameterless member whose whole body is a field test of `this`. A **helper predicate** is an exported one-parameter function doing the same for its parameter. A **selector** is a function whose result is a boolean of tests on its parameters, or a type predicate.
- **Outcome**: what an arm produces: whether it runs **commands**, and what its result depends on: none, empty, constant, the value, parts of it, other data, or a mix. A constant result without commands is a **lookup**.
- **Command**: something an arm runs for what it does rather than for a value: an expression statement whose value is discarded (a call, an update, an assignment to data other than a local of its function), or a returned call typed `void`. This is not purity: `const r = await save(x)` is not a command, and `list.push(x)` on a fresh local array is.
- **Control flow** of an arm: the `if`, `switch`, ternary, loop and `try` nodes written in it.
- **Decision**: a place choosing between arms on one value, each guarded by tests of its literal values. A branch's claim is **complete** when its conditions test only that value and do not overlap earlier arms.
- **Site**: a written question about vocabularies (a condition, `case` labels, or a typed list), indexed per program by the literal values it mentions.

## Choices

- Types only: plain `string` values get no vocabulary guessed from the literals they are compared with.
- Every condition is read one way, as tests of a value with a claim.
- Sites compare outcomes as text, because the shared index is built without a checker.

## Limitations

- Returns are followed only to call sites in the same file, not through callbacks.
- A `throw` inside a `try` with a `catch` is not a rejection.
- A truthiness test of a `boolean` treats `false` as missing.
- A variable naming data is not updated when that data changes through another path.
- Values declared only in declaration files, such as `JSON` or `process.env`, count as code, not data.

Facts are pinned by the `model-facts` probe fixtures; see [native](../native.md#shared-decision-model).
