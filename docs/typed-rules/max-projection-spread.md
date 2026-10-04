# max-projection-spread

## Intent

Limit how many files restate how an owned value becomes one of its projections. A projection is an object literal built from one value of a project type: `{ id: site.id.serialize(), private: site.getVisibility().is("private") }`. Each property restates a derivation: which members of the value produce it, and how. When several files build projections of the same type by hand, each derivation they share is restated in each of them. Changing what `private` means, or how identifiers are serialized, requires finding every file by hand, and the compiler points to none that is missed: a named response type checks each site's shape, not how each property is derived.

The cost grows with the number of files, not the number of projections: projections concentrated in one place, such as an owner method or a dedicated mapper, count as one file.

The diagnostic describes the spread. It does not prescribe a remedy: an owner method, a mapper shared by the sites, or deliberately separate copies (such as a versioned format that must not follow later changes) are all possible designs.

Status: experimental research candidate. Not ready for promotion: see [Evaluation](#evaluation).

Oxlint 1.83.0 has no rule that compares object construction across files. `max-interpretation-spread` measures a different restated knowledge: checks of a closed vocabulary's values.

## Runtime and options

Native Go rule, run by Oxlint through the `aexlint/typed-plugin` JavaScript plugin, which delegates to the bundled backend.

```json
{
  "jsPlugins": ["aexlint/typed-plugin"],
  "rules": {
    "aexlint-typed/max-projection-spread": ["warn", { "max": 2 }]
  }
}
```

`max` is optional and must be a positive safe integer: the greatest number of files, including the linted one, that may restate one derivation. It defaults to 2, also when the rule is enabled without options. Unknown properties, non-object options, and non-integer or non-positive values produce a backend configuration error. The default is not a validated boundary.

## Measurement

**Derivation.** A property assignment in an object literal, with a static key (an identifier, string, or number), whose value is a chain from an identifier through property accesses and method calls, with at least one call, and whose call arguments are all literals (strings, numbers, booleans, `null`) or absent. Parentheses, `as`, `satisfies`, and non-null assertions are looked through. `id: site.id.serialize()` and `private: site.getVisibility().is("private")` are derivations; `id: site.id` (a copy), `name` (shorthand), `label: format(site)` (a function of the value, not a member chain), and `name: site.getName(locale)` (a non-literal argument) are not.

**Owner.** The declared type of the chain's root identifier, with `null` and `undefined` removed: its type alias, otherwise its symbol, which must be declared in project source files. Values of library types (`Date`, `Map`) and of types without a symbol are not owners.

**Projection.** An object literal with at least two derivations whose roots are the same variable with the same owner. A single derivation in a literal is a use of the value, not a projection of it.

**Identity.** Two derivations restate the same derivation when they have the same key, the same owner, the same members along the chain resolved by the checker to the same declarations, and the same literal arguments. Text equality is not used: members are compared by their declarations, so a chain through a differently named alias of the same value is the same derivation, and same-named members of different types are not.

**Count.** The number of distinct files containing a projection that restates a derivation, searched across the whole TypeScript program containing the linted file, including files that are not being linted. Test files that belong to the program count like any other file.

## Reported code

```ts
// get-site.ts
return {
  id: site.id.serialize(),
  name: site.getName(),
  private: site.getVisibility().is("private"),
};

// list-sites.ts, search-sites.ts
sites.map((site) => ({
  id: site.id.serialize(),
  private: site.getVisibility().is("private"),
}));
```

With the default limit of 2, `id` and `private` are restated in 3 files. Each of the three files reports `spreadProjection` once per projection, on the key of its first spread derivation:

```text
This projection of `Site` restates 2 derivations that other files also restate (maximum 2 files each): `id` (`id.serialize()`) in 3 files, `private` (`getVisibility().is("private")`) in 3 files.
help: Changing one of these derivations requires finding each file by hand. Other files, nearest first: list-sites.ts, search-sites.ts.
```

The message lists up to three spread derivations, most files first; the help lists up to three other files, nearest first (most shared leading directories, then path). Paths are relative to Oxlint's working directory.

## Accepted code

```ts
// site.ts: the owner builds its projection once.
class Site {
  toResponse() {
    return { id: this.id.serialize(), private: this.visibility.is("private") };
  }
}

// get-site.ts, list-sites.ts, search-sites.ts
return site.toResponse();
```

Also accepted:

- projections whose derivations are restated in at most `max` files;
- object literals with one derivation, copies (`id: site.id`), and shorthand properties;
- derivations rooted at `this`, which belong to the owner itself;
- derivations of library types, such as `{ at: date.toISOString(), day: date.getDay() }`;
- derivations with non-literal arguments.

Cross-file example: the linted file and one other file project `site.getVisibility().is("private")`. Adding a third file with the same projection makes all three report; changing only the third file's projection to `site.getVisibility().is("public")` makes none report.

## Evaluation

Labeled evaluation, 2026-10-04, default limit, on an application backend of about 4,600 files; no reports on two smaller tools or on this repository. Every report was labeled; members of large clusters (versioned serializers, test doubles) were labeled from representative members. Per-report labels are kept outside the repository.

| Label          | Reports | Patterns                                                                                                                                                                                                                                                                                                                                         |
| -------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Useful         | 12      | Whole objects rebuilt by hand in several files: a domain event assembled in three commands whose copies already disagree on one field, an export row written out in two places, a search-index entry rebuilt by a maintenance script instead of the owner's factory and already different from it, a public response shape shared by two queries |
| Debatable      | 7       | The owner's own factory reported as one of the copies; copy-pasted loaders that may or may not be one intended contract                                                                                                                                                                                                                          |
| False positive | 67      | Versioned serializers (47), test code (10), single shared trivial derivations (4), separate read models sharing a few getters (3), coincidental external contracts (2), other (1)                                                                                                                                                                |

Findings:

- **Per-derivation counting finds useful cases by accident.** Every derivation over the limit was a single zero-argument call (`getName()`, `id.serialize()`). The useful findings were near-total copies of a whole object, found through such getters; the one derivation carrying real knowledge (`getVisibility().is("private")`) stayed under the limit.
- **Versioned serializers dominate.** They share two or more derivations each, so requiring several shared derivations does not exclude them. A principled exclusion is needed; file-name patterns are not acceptable.
- **Similarity of whole projections**, the share of identical key-to-derivation pairs between two literals, separates the useful cases (near-total matches) from read models (2–4 shared getters out of 10–20 keys) better than per-derivation file counts. This is inferred from the labels and untested.

Precision is about 14% counting only useful reports. The measurement unit needs to change before the candidate can be evaluated for promotion.

## Fixes

Diagnostic-only. Moving projections to one place is a design decision.

## Limitations

- **Deliberate copies are reported.** Versioned serializers, read models, and projections into different external contracts duplicate derivations on purpose. They are reported like accidental copies; suppress them or raise `max`.
- **Only member chains are derivations.** A projection through helper functions (`formatDate(site.createdAt)`), through intermediate variables, or with computed arguments is not compared.
- **Per-property identity.** Common derivations such as `id: x.id.serialize()` may be shared by many otherwise different projections and dominate the counts.
- **Only object literals.** Projections built by assignment to a fresh object, or by class constructors, are not seen.
- **Cost.** The first linted file in a program builds a syntactic index of candidate derivations in every project source file. Each invocation resolves, with its own checker, the candidates whose fingerprint matches a derivation in the linted file, and their sibling properties.

Requires a TypeScript 7-compatible tsconfig.json. TypeScript's own errors are left to `tsc`. Editor integration is not supported yet.
