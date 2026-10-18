# no-widened-union

Reports a `string` parameter or result compared with literals while only values of a literal union reach it. The union is lost where it is widened: a misspelled comparison compiles, a value added to the union silently takes the fallback, and rules that read unions cannot see the comparisons.

```json
{ "rules": { "aexlint/no-widened-union": "warn" } }
```

No options.

## Example

Reported on `string | undefined`, since every call passes an `Effort`:

```ts
type Effort = "low" | "high" | "max";
function indicator(level: string | undefined) {
  switch (level) {
    case "low":
      return "○";
    case "high":
      return "●";
  }
  return "";
}
indicator(settings.effort);
```

Reported on `: string`, since every return gives an `Effort` and a caller compares the result:

```ts
function effortOf(task: Task): string {
  return task.urgent ? "max" : task.effort;
}
if (effortOf(task) === "max") alert();
```

Accepted: `level: Effort | undefined`, a call passing a plain `string`, or a result nobody compares.

## What counts

- A named function with a body: a function declaration, a function or arrow initializing a variable or class field, or a method. Type predicates are skipped: they are meant to accept the wider type.
- **Parameters:** a parameter with a written type that is `string`, possibly with `null` or `undefined`, compared with a string literal in the function: an equality, a `case` label of a `switch` on it, or `includes` on a literal list. Every call in the program passes string literal types for it, and at least one passes a value whose type holds several: literals written at the calls alone are values, not a union. `null`, `undefined` and calls leaving it out are ignored.
- **Results:** a written result type that is `string`, possibly with `null` or `undefined`, on a function that is neither async nor a generator. Every returned expression has string literal types, together at least two values, and a call compares the result with a literal: directly, through a `const` it initializes, or by passing it to a parameter its function compares.
- Types are read where values are passed or returned, so a literal written there counts, and a generic value gives its constraint.
- A call through a type the method's class matches, such as an interface it implements or matches, or a class with the same members, counts as a call to the method.
- A parameter is reported only when the function is used by being called. Passing or storing it, a default export, renaming it on import or export, destructuring it, overloads, and a method that a base class also declares make other callers possible, so the parameter is skipped.

## Reporting and limits

Reports `widenedUnion` on the parameter's type and `widenedResult` on the result type, naming the values. No fixes: the union may need a name first.

Calls on `any`, through element access, or from outside the program, such as consumers of a published package, are not seen. A parameter of generic, `any` or `unknown` type is not `string`.
