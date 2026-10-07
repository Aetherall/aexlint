# Developing aexlint

Install [devenv](https://devenv.sh/getting-started/), then:

```sh
devenv shell
pnpm install --frozen-lockfile
pnpm check
```

`devenv.nix` provides Node.js 24, Go 1.26, pnpm, tar, and patch. `devenv.lock` pins Nix inputs; `packageManager` pins pnpm; `pnpm-lock.yaml` pins JS dependencies. Shell entry does not install dependencies or run checks. The optional `.envrc` supports direnv after review and approval.

The first native build downloads checksum-verified upstream source archives and Go dependencies. Subsequent builds reuse `.native/` and Go's compiler/module caches. No global installation, git commits, submodule changes, or elevated commands are involved.

## Linting aexlint with its own rules

`pnpm lint` checks `src`, `scripts`, and `tests` with both runtimes:

- `.oxlintrc.json` loads the current TypeScript plugin source directly, with workload max **4**, input depth max **3**, and decision depth max **2**.
- `.oxlintrc.typed.json` loads the typed plugin source and enables `aexlint-typed/prefer-truthy-presence-check` as an error against the repository's real TypeScript project. The script points the plugin at the freshly built backend with `AEXLINT_TYPED_BACKEND`.

The command rebuilds the host native backend before the typed pass, so it works without a prior package build and does not silently use stale rules. This requires the development Go toolchain and the same first-build downloads described above. Native preparation/builds must not run concurrently with `pnpm lint`.

Both passes run even when the JS pass reports violations. Either pass's errors fail the command; existing Oxlint warnings also fail. These are this repository's explicit limits, not a consumer preset. Findings are not suppressed or automatically fixed. Test fixture strings are data; the tests and helpers themselves are linted.

`pnpm lint:classifications` separately runs the experimental `prefer-domain-predicate` rule as warnings. Pass a file or directory to narrow the scan, or use `--format json` for repeatable review. Classifications may be named through a result, category data, or a predicate/validator; a one-off check does not require a helper. Anonymous combinations of representation checks still report. Read the [rule contract](rules/prefer-domain-predicate.md) before acting on reports.

## Syntax/scope rules

```sh
pnpm rule:new no-example
pnpm test:rule src/rules/no-example/no-example.test.ts
```

Rules use `defineRule` and Oxlint's real `RuleTester`. Focused tests run TypeScript source directly without rebuilding.

## Type-aware rules

```sh
pnpm rule:new:typed no-typed-example
pnpm native:test no-typed-example
pnpm native:test no-typed-example --update
```

Rules use the upstream Go `rule.Rule`, `rule.RuleContext`, and `rule_tester.RunRuleTester`. The context provides the real `Program` and `TypeChecker`. The build automatically registers each `native/rules/<name>/rule.go` export named `Rule`; no manual registry edits are needed.

Both generators create **intentionally failing placeholders**, not working product rules. Define the contract and replace the examples. Go snapshots are retained in `native/snapshots/`; review changes when using `--update`. Normal tests fail if new snapshots would be generated rather than silently accepting them.

See [AGENTS.md](../AGENTS.md) for the agent workflow and [native architecture](native.md) for source pinning, integration limits, and upgrades.

## Commands

| Command                            | Purpose                                                                                          |
| ---------------------------------- | ------------------------------------------------------------------------------------------------ |
| `pnpm test:rule <path>`            | Focused JS rule tests; add `--watch` for watch mode                                              |
| `pnpm native:test [name]`          | Real Go rule tests, including the infrastructure type probe                                      |
| `pnpm native:test [name] --update` | Explicitly update Go diagnostic snapshots                                                        |
| `pnpm typecheck`                   | Strict TypeScript checks for source, tests, and scripts                                          |
| `pnpm lint`                        | Oxlint plus stable aexlint rules on source, scripts, and tests; rebuilds the host native backend |
| `pnpm lint:classifications [path]` | Experimental domain-classification diagnostics; warnings only, no native build                   |
| `pnpm format`                      | Oxfmt and gofmt                                                                                  |
| `pnpm build`                       | Clean ESM/declaration build and host native executable                                           |
| `pnpm native:build`                | Rebuild only the host native executable                                                          |
| `pnpm test`                        | Host build, native rule tests, TS tests, and packed-package validation                           |
| `pnpm check`                       | Typecheck, lint, formatting, and all tests                                                       |
| `devenv test`                      | Frozen pnpm install and full checks inside devenv                                                |
| `pnpm build:release`               | Build all six native targets into the same package                                               |
| `pnpm pack`                        | Release build and one tarball; does not publish                                                  |
| `pnpm package:smoke <dir>`         | Install the one tarball in `<dir>` into a fresh consumer and run both plugins on this host       |

CI (`.github/workflows/ci.yml`) runs the checks on every push and pull request and cross-compiles a release tarball. It then installs that same tarball on Linux, macOS and Windows runners, each on x64 and arm64, and runs `pnpm package:smoke`, which lints one file with both plugins and so executes the bundled native backend on each platform. The smoke test checks that each executable starts and reports; the full rule suites run only on the Linux x64 host.

## Why a second plugin?

Oxlint's JS plugin API does not expose type information. Its built-in type-aware rules are compiled into the separate Go backend, `tsgolint`, and Oxlint forwards only its own compiled rule names to it: changing `OXLINT_TSGOLINT_PATH` alone cannot register our custom names.

We therefore reuse pinned `tsgolint` and `typescript-go` code through a small native build overlay, and run it from an ordinary Oxlint JS plugin that delegates each TypeScript program to that backend. There is no fake JS type checker, ESLint fallback, or fork of the Oxlint frontend.

- [Oxlint JS API support](https://oxc.rs/docs/guide/usage/linter/js-plugins#api-support)
- [Oxlint type-aware architecture](https://oxc.rs/docs/guide/usage/linter/type-aware)
- [tsgolint source and rule-authoring APIs](https://github.com/oxc-project/tsgolint)

## Packaging and first release

One tarball contains all JS rules plus native executables for Linux, macOS, and Windows, each on x64 and arm64. Go builds use `CGO_ENABLED=0`; no postinstall downloads or build scripts are required. Bundling six executables makes this larger than a JS-only plugin.

Test-only probe rules, Go source caches, unit tests, and development scripts are excluded. Third-party native license/notice texts and the pinned source revisions are included under `dist/native/`. `dist/` and `.native/` are generated, disposable directories.

## Releasing

`.github/workflows/release.yml` publishes to npm when a `v*` tag is pushed:

1. It fails unless the tag equals `v` plus the `version` in `package.json`.
2. It runs the whole CI workflow, including the six-platform smoke tests.
3. It publishes the exact tarball CI tested with `npm publish --provenance`, from the `npm` GitHub environment.

Publishing uses npm trusted publishing (GitHub OIDC), so the repository holds no npm token. On npmjs.com, the `aexlint` package's trusted publisher must name repository `Aetherall/aexlint`, workflow `release.yml` and environment `npm`. npm only accepts that setting for a package that already exists.

For the first publication, choose the release version in `package.json`, commit and push the branch without a release tag, and wait for all six CI smoke tests to pass. Download the `package` artifact from that successful run and publish its tarball manually using an authorized npm account. Configure the trusted publisher before using the tag-triggered workflow for subsequent versions. Do not rebuild the first-release tarball after CI validation or try to publish that same version again through a tag.

For subsequent releases, update `version` in `package.json`, commit, then tag and push:

```sh
git tag v0.1.0
git push origin master v0.1.0
```

Review the rule contracts, third-party notices and packed contents before each release.
