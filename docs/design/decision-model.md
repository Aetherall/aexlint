# Decision model

**Status: shared Go package `native/model/`, used by the typed rules listed under [rules as queries](#rules-as-queries). Conflicting lookups are tested through a test-only probe and used by no rule.**

## Goal

Code knows things about values. A **fact** is established by a test, carried while it holds, and used. The smells the rules find are facts that are lost and rebuilt, established twice, established outside their owner, or spread across files. The model represents each part of a fact once, from real types and syntax, so precision is fixed in one place and rules are short queries that combine the parts.

A fact has six parts: the [value](#value) it is about, the [claim](#claim) it makes, the [test](#test) that establishes it, [where it holds](#where-a-fact-holds), including across [boundaries](#boundaries), the [owner](#owner) who should hold it, and the [outcome](#outcome) of the code it guards.

## Value

A **value** is data reached from a root through property reads, optional or not, and element reads with a literal key. The root is a variable, `this` in one function or class, or an expression evaluated once, such as a call. Two reads are the same value when their roots and paths are the same, so two calls written alike are different values. Identity comes from symbols, not text. A value is **partial** when it stands for an unknown part of the data at its path, such as what a method call reads.

A **field** is what a value names across code: the declarations of the property it ends at, or of the variable it is. A variable bound by destructuring, as in `const { kind } = item` or `function f({ kind }: Item)`, names the property it binds. A property is looked up on its object's declared type, so narrowing does not change it, or on the narrowed type when only some members of the declared type have it. Two values are the same field when they have the same name and share a declaration.

Code is compared by its whole written text, with whitespace collapsed; shortened text is only for display.

A **form** is a structural comparison key for an expression relative to a source value, not proof of computational equivalence. The source becomes `$`; member and literal-key reads keep their paths and optional-access operators, with dot reads and equivalent string-key reads normalized together. Literals are compared by value and kind, and calls by their callee, arguments, and optional-call operator. External references use resolved symbol identity, including imported aliases; bindings declared inside an expression keep their written names. Object entries are reordered only when their keys are static and unique and their initializers contain only known, non-accessor data reads or supported literal and boolean expressions. Spreads, computed keys, duplicate keys, accessors, and calls preserve entry order. Other code keeps its written structure with source reads and external references replaced.

## Claim

A **claim** is what a test establishes about a value: it is one of a set of values, among a domain. Claims compare as sets: the same, the opposite, or one implying another. Restricted to what a type can hold, a claim tells what a test changes there.

- A claim about **literal values** ranges over the value's **vocabulary**: the finite literal values of its declared type, at least two string or number literals once `null` and `undefined` are removed. They are compared with literals, constants, enum members, or properties declared with a single literal type, such as members of an `as const` object.
- A claim about **presence** ranges over present, `undefined`, `null`, and other falsy values. See [presence](#presence).

### Presence

A **presence test** checks whether a value is there. It is a truthiness test of a variable, property or element read (`x`, `!x` in a condition), or a comparison of any expression with `null` or `undefined` (`x == null`, `f() !== undefined`). It **removes** missing values: `undefined`, `null`, and for truthiness also other falsy values such as `""`, `0` and `false`. Which of them the value can hold comes from its type at the test; `any` and `unknown` can hold all of them. A type holds no other falsy value when every member is truthy: a non-empty string, a number other than `0` and `NaN`, a bigint other than `0`, `true`, a symbol, or an object type that admits no primitive and is not `document.all`. `{}` and `Object` admit primitives, so they can be falsy. An intersection with a primitive member holds that primitive's values, so `string & { brand: "id" }` can be `""`; an intersection of object types is truthy when it admits no primitive. A type whose members are all brands, keyed by unique symbols, describes no runtime shape: opaque types use it to stand for primitives, so it can be falsy.

A function **rejects** a missing parameter when one of its [entry tests](#where-a-fact-holds) is a presence test of the parameter itself, leaves when the value is missing, and has a fallback: `undefined`, `null`, a boolean, a string or number, `[]`, or `{}`. This is the [rejection](#rejection) of a missing value: the branch does not use it and returns nothing new.

The model can also work out a test's result when a value is **missing**, from `&&`, `||`, `!`, `Boolean(…)` and comparisons with literals, so a suggested call can be checked to give the same result.

## Test

A **test** is a place where code learns a claim about a value: the claim holds where the test's expression is true. It is a comparison with literal values, a membership test with `includes` on an array whose values are known exactly (an array literal, or a tuple of literal types such as an array written `as const`), or a presence test. A bare read is a test only where its value is used as a **condition**: by `if`, a loop, a conditional expression, `!`, or as an operand of `&&` and `||` that decides. A value used **only as a boolean** is a condition whose value never escapes, so `a !== undefined && b`, whose result is used, is a condition but not only a boolean.

A **check** is a place that rejects input without the model reading what it establishes: a `throw` under conditions that read input, or a call to a function that checks what it receives. It is known by the data its conditions read. Tests that only establish what TypeScript keeps in a narrowed type (presence, `typeof`, `instanceof`, `in`, type guards) are not checks.

## Where a fact holds

### Rejection

A branch **rejects** variants when it does not read the tested value and the other side does. Reading the tested subject itself to test it again does not count as using the value, nor does mentioning it on the rejecting side, for example in an error. Passing it on, or reading it otherwise on the other side, does. For `Item = TextItem | StockItem`, with kind `"text"` or `"stock"` and stock sku `"alpha" | "beta"`, `if (item.kind !== "stock") return undefined;` rejects non-stock items when the rest uses `item`: the return ignores it. `if (item.kind === "stock") return item.sku === "alpha" ? "available" : "reserved";` does not reject: the branch reads `item`, so it is work. `if (item.kind === "text") return { label: "Text" }; return undefined;` does not reject either: neither side reads `item`.

A **guard** is a test after which code does work only where a claim holds, and otherwise leaves without using the tested variable: it is a rejection. It is written as one of:

- `a && b ? work : exit` or `a || b ? exit : work`: each of `a` and `b` is a guard;
- `a && b && work`: `a` and `b` leave with their own value when they fail. When the chain is itself a condition, its operands that are tests are part of the condition, not work. That value is `false` for a `boolean` test, and the missing value for a truthiness test whose type allows only one missing value. Each shorter prefix is read the same way, so in `a && b && c`, `a` also guards `b`.

An **entry test** is a guard a function makes on a parameter, or on a property read from it, before its work. Its forms are:

- a first statement `if (a || b) exit;` without `else`, where `exit` returns or throws: the function leaves when `a` or `b` holds;
- a single statement `if (a && b) { work }` without `else`: the function leaves when `a` or `b` fails, and returns nothing;
- a guard written as the body, the returned value, or the single expression statement.

The **fallback** of an entry test is the constant the function returns when it leaves: the returned constant, the value a guard leaves with, or `undefined` when the function then returns nothing. A single expression statement returns nothing either way, so it leaves with `undefined` when its exit is a constant. There is none for async functions and generators, whose results are wrapped.

A later read **relies** on a presence test when its type there no longer holds a missing value the test splits off; the test then narrows for that read, and is not only a question. A check **precedes** a call when the statement holding it comes before the call in a statement list enclosing both, and no list between them ends by leaving it. Two places are **exclusive** when the `if` conditions guarding them below the statement list enclosing both make claims about the same value that no value satisfies together: tests joined by `&&` hold in a `then` branch, and the opposites of tests joined by `||` in an `else` branch. Two decisions are in **exclusive branches** when they sit on different sides of the same `if`, conditional expression, or `switch` clause, counting code after an `if` whose `then` always exits as its `else` side.

A function's **flow** is what it does with its input, its parameters and the locals derived from them: the data each value is built from, through aliases, locals and stores, the checks it makes, the calls it passes data to, and the data it returns.

### Boundaries

A **boundary** is where a value passes from one function to another. Three questions can be asked there: what is known about the value on the way in, whether the receiving side's declared type keeps it, and whether the receiving side checks it again. Knowledge lost at a boundary and checked again is checked twice. Knowledge checked only on the way in is checked outside the code that receives the value.

The model reads four kinds of boundary:

- An **argument** fills a parameter of the called function. The function comes from the call's resolved signature and needs a body in a project file. A `this` parameter is skipped. An argument after a spread, and a rest parameter, have no boundary. What the receiving side checks again is its entry tests. The parameter **keeps** what a test establishes about the argument when the argument as written, without the test's narrowing, is not assignable to the parameter's type in the resolved signature: passing it untested would not type-check. A cast changes the argument's written type. A default value replaces `undefined` as it **arrives**, so the receiving side never sees it.
- A call's result carries the arguments the function's return statements read: the **return** boundary. A result type **keeps** a check when only code with access to its private state or brand can produce it: a project class or an object type with private, protected, `#private` or unique symbol members.
- A constructor or method **stores** a parameter into its class's state, by assigning it to a property of `this`, passing it to a member that stores it, or, for a static factory, reading it at all: the **field** boundary.
- A **gate** is a selector and a handler in the same object or class whose first parameters have the same type. Code calling the handler with a value the selector accepted passes the value across the gate.

A function whose body, returned value, or single expression statement is a call passing its parameters on, to a project function with a body, has the entry tests of that function across the boundary.

## Owner

An **owner** is who a type belongs to: a class, a runtime value, or a declaring module or namespace, resolved as described in [variant behavior ownership](variant-behavior-ownership.md). A type made by an alias mapping over `keyof` one of its type parameters, such as `Readonly<T>` or `Partial<T>`, belongs to the owner of the type it maps unless the alias is itself owned. Code is **inside an owner** when it is written in the owner's own declaration, or for a type-only owner, in its module. Code is **inside a member's owner** when it is written in a class that has the member, declared there or inherited, mixins included. A class **extends** another when it is that class or derives from it through `extends`, at any depth.

An owner answers questions about its values through **answers**: functions whose result is a test.

### Predicates

A **field test** is a test of a project property whose claim, restricted to what the property's declared type can hold, splits it. The property is resolved where it is read, so on a narrowed receiver its declared type is the narrowed member's; which field it is still comes from the receiver's declared type. Two field tests **ask the same question** when they test the same property declaration, write the same literal values, and split the values both declared types can hold the same way, in either direction: a generic property and its instantiation are compared on what both can hold. A **predicate** is a method, getter, or arrow-function property without parameters whose whole body is a field test of `this`. The type that declares it owns that question; a field test elsewhere asking it about a value of that type rewrites it. A **helper predicate** is an exported function of one parameter whose whole body is a field test of that parameter. Helper modules stand in for ownership when classes do not fit, or when the type should not depend on its behavior, as with traits. A function **declares** that it narrows when its written return type is a type predicate, `x is T`.

A **selector** is a function whose whole result is a boolean made of tests of literal values on its parameters, or that declares a type predicate. Its **answer** is the tests that hold where it returns true: joined with `&&`, merged across `||` on one value, negated with `!`, and read through calls to project functions with a body, in any file of the program, up to three calls deep, across their boundaries.

## Outcome

An **outcome** is what an arm produces when it runs: the data its result depends on, and whether it has **effects**, doing more than producing it. The result is the arm's expression, or the value its last statement returns; an arm whose result is a call returning `void` or `undefined` has effects and no result. Relative to the value its decision tests, or to the receiver when it tests a property, the result depends on:

- **none**: there is no result, as with `return`, `throw`, `continue` or statements only;
- **empty**: nothing new, as with `undefined`, `null`, a boolean, `""`, `0`, `-1`, or object and array literals holding only missing or empty values;
- **constant**: constants only, compared by their literals or their form;
- **value**: the value as a whole: the arm returns it, passes it on, or calls a method on it;
- **parts**: parts of the value other than the tested property;
- **other**: other data only;
- **mixed**: the value and other data.

What code **reads** relative to a value is the value as a whole, the value's fields other than the tested one, and other data; outcomes and constructions are both read this way. Functions, classes, namespaces, enums and ambient globals are code, not data, and so is a function called by name or a method of `this`; the receiver of any other call is data, and calling a method of the value reads the whole value. Functions written in the code do not run with it. Constants and bindings declared in the arm read nothing. An arm does no **work** when it has no effects and its result is none or empty, and it is a **lookup** when it has no effects and its result is constant. The outcome describes what runs: a conditional expression is part of a value, and functions written in the arm do not run with it.

**Logic** is what an arm contains: the `if`, `switch`, conditional expression, loop and `try` nodes written in it, in source order, including inside arrow functions and function expressions written there, but not inside named nested functions or methods. Logic describes what is written, whether or not it runs with the arm.

A lookup's **purpose** is the **slot** its result fills:

- a call parameter, continued through the call when the call's own result fills another slot, but not into a function's return;
- an object property, qualified by the object's contextual type;
- the return of one specific function, together with the purposes of that function's call sites in the same file;
- a local variable without a type annotation is followed to its uses.

A bare contextual type such as `string` or `ReactNode` is not a slot. A chain ending in one has no known purpose.

## Structures

The parts combine into the structures rules query:

- A **decision** is a place where code chooses between arms on one value, each arm guarded by a test of its literal values, and holding the claim true where it runs. A branch's claim is what its condition implies about the value: other operands of `&&` and `||` establish nothing about it, so in `x === "a" || (x === "b" && ready)` the branch holds for `"a"` and `"b"`. Values that an earlier statement of the same list has left on, through an `if` whose whole condition tests the value and whose `then` exits, are in no branch. A branch records whether its claim is **complete**: its condition and preceding conditions contain only tests of that value, and its accepted variants do not overlap preceding arms. Otherwise the claim is necessary evidence, not a complete partition. Construction and conflicting-lookup comparisons require complete claims in every arm. The arms of one decision test the same written expression. Forms: guard (at most one arm does work) and dispatch (several arms do work). A decision on a property records its receiver, the owner of the receiver's type, and whether it is written inside that owner.
- A **domain** is an entry test of a function on literal values, possibly reached through a forwarding call, with who narrows it. A domain is narrowed by **its own guard** unless a selector gating it answers with a test of the same part of its first parameter whose claim implies the domain's. Then the selector owns the narrowing, and the guard repeats it. Calls to a guard or selector are reuse, not new decisions.
- A **site** is a written question about vocabularies: a condition, a group of `case` labels, or a typed list of values, with the tests it makes. The shared index lists the sites of a program's project files by the literal values they mention; a site comparing with anything else, such as a constant or an enum member, is listed apart, since only a checker knows its values.

## Relations

- **Same** and **Implies**: claims about the same value compare as sets: the same values accepted, or one claim's values inside another's.
- **Conflicts**: two lookup dispatches on the same field map a variant to disjoint values, share a purpose, and are not in exclusive branches.

## Rules as queries

- `no-repeated-presence-check`: a guard whose work is one call, across whose argument boundary the parameter does not keep the claim and an entry test rejects again every value the caller rejects, with the same fallback.
- `no-repeated-selection`: a domain narrowed by a selector across a gate.
- `no-overbroad-type-predicate`: a type predicate whose body rejects values of its declared target.
- `no-fact-outside-owner`: a field test written outside the owner of a predicate or helper asking the same question, whose narrowing no later read relies on; or a property whose field tests ask the same question about values it can hold in more than a limit of files outside its owner, when no predicate or helper answers it.
- `prefer-truthy-presence-check`: a comparison with `undefined` whose claim, under the type at the comparison, splits values like truthiness.
- `no-procedural-value-dispatch`: a decision whose tested arms contain logic.
- `no-external-variant-behavior`: a decision on a property, outside its owner, whose arms construct results from the receiver's other fields.
- `no-repeated-conversion`: decisions on the same field whose arms build the same constructions.
- `max-interpretation-spread`: the files whose sites check a vocabulary.
- `no-repeated-value-group` and `no-repeated-value-policy`: sites asking the same question, or the same question with the same outcome, in several places.
- `no-forgeable-validated-result`: a check in a function's flow on data it returns through a result type that does not keep the check.
- `no-caller-enforced-invariant`: a check preceding a call that hands its data to a member storing it across the field boundary without checking it.

## Decisions

- **Types only.** A decision's field needs a finite literal type. Values typed as plain `string` are not given a vocabulary guessed from the literals they are compared with; typing them as a union is the fix that makes them visible.
- **Conflicts are not a rule yet.** The relation stays in the model; its usefulness and generality remain unestablished.
- **One reading of conditions.** Sites, decision arms, guards, entry tests and selector answers are all built from tests: a value and a claim. A claim about presence ranges over its own vocabulary, so every claim has one.
- **Kept apart on purpose.** A site's outcome is written text, because the shared index that keys it is built without a checker. A type predicate's contract compares with its target type, including open `string`, which no finite claim can range over. A check's claim is not read; it is known by its data.
- **Grouping is by written expression.** The arms of one decision, and the tests merged in one site, test the same written expression: `getKind() === "a" || getKind() === "b"` is one question about values, although two calls are two pieces of data.

## Limitations

- Returns are followed only to call sites in the same file, and not through callbacks such as `useMemo(() => ...)`. Recall is unknown.
- Fields need a finite literal type. Tests on open `string` values are invisible.
- Selector tests expand calls to project functions with a body, up to three calls deep, and merge `||` alternatives on the same field. Declared functions without a body, and entry guards written as calls, are not expanded; a call in a guard is reuse.
- Boundaries carry claims only from arguments to parameters and across gates; the return and field boundaries carry checks, known by their data.
- JSX attributes and children are not slots.
- Selectors and handlers are related only inside one object literal or class.
- Field identity across files relies on property declarations; aliases through mapped or generic types are unverified.
- An array declared with an array type, even `readonly`, only says which values it may hold, so `includes` on it is not a test; write the array `as const`. Earlier exits narrow a decision only from the same statement list, and a value reassigned in between is not noticed.
- Flows do not read checks made in callbacks a function passes on, such as a `throw` inside a `filter` callback: functions written in code do not run with it.
- Outcomes treat values declared only in declaration files as code, so data they export is not counted, and a read with a computed key counts as reading the whole value.
