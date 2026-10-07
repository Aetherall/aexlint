# max-projection-spread

Limit how many files repeat how a project-owned value becomes an output object. Repeated member calls can leave several places to update when the meaning of an output field changes.

**Status:** experimental research candidate; not ready for promotion. Diagnostic-only, with no fixes or suggestions. Evaluation found mostly deliberate copies or unrelated projections sharing getters, not useful design problems.

## Enable

```json
{
  "jsPlugins": ["@aetherall/aexlint/typed-plugin"],
  "rules": { "aexlint-typed/max-projection-spread": ["warn", { "max": 2 }] }
}
```

**Options:** optional `max`, a positive safe integer; default **2**. Unknown keys and invalid values are configuration errors. The default is not a validated design boundary.

See [typed-plugin setup](../../README.md#type-aware-setup-and-limits) for project requirements. Run `tsc` separately; editor integration is unsupported.

## Example

Reported when three files build this projection of the same project `Site` type:

```ts
return {
  id: site.id.serialize(),
  private: site.getVisibility().is("private"),
};
```

Accepted alternative, when these sites should share one representation: define the projection on `Site` and call it from each site.

```ts
toResponse() {
  return {
    id: this.id.serialize(),
    private: this.getVisibility().is("private"),
  };
}
```

The callers then use `site.toResponse()` rather than rebuilding the object.

## What counts

- A derivation is an object-literal property with a static key, built from an identifier through member accesses and at least one method call. Call arguments must be literals (string, number, boolean or `null`) or absent.
- The root's non-nullable object type must have an alias or symbol declared entirely in project source files. Library types and types without that identity are excluded.
- A projection needs at least two derivations from the same variable and owner type. A single method call in an object is not enough.
- Derivations match when the output key, owner, resolved member declarations and literal arguments match. Variable spelling alone does not determine identity.
- Each derivation counts distinct files, not whole-object copies. The search covers the TypeScript program, including unlinted source files and tests, but not declaration files or external libraries.

## Reporting scope

Above `max`, reports `spreadProjection` once per projection, on the key of its first over-limit derivation. The message names repeated derivations and their file counts; help lists other files.

Copies such as `id: site.id`, shorthand properties, roots at `this`, calls to standalone helpers, and calls with nonliteral arguments do not count. Projections within the limit are accepted.

## Limitations

- Versioned serializers, separate read models and external formats may repeat derivations deliberately. A report does not prove they should share code.
- Common getters can exceed the limit even when the surrounding objects differ substantially. The rule does not measure similarity of whole objects.
- Intermediate variables, helper functions, computed arguments, assignments into fresh objects and constructor-built projections are not compared.
- Other tsconfig programs are not searched. Program-wide indexing and per-file type resolution add cost on large projects.
- Earlier evaluation found only about 14% of reports useful. The measurement needs further research before promotion; raising the limit or suppressing deliberate copies may be appropriate.
