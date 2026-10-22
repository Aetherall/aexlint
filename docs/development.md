# Development

## Setup

Install [devenv](https://devenv.sh/getting-started/), then:

```sh
devenv shell
pnpm install --frozen-lockfile
pnpm check
```

The shell provides Node.js, pnpm, and Go. The first native build needs network access; later builds reuse `.native/`. Keep the lockfiles intact. Do not run native builds, tests, or self-lint concurrently.

## Everyday commands

| Command                        | Purpose                                                              |
| ------------------------------ | -------------------------------------------------------------------- |
| `pnpm native:test [--update]`  | Run the model probes; review snapshot changes                        |
| `pnpm test:project [--update]` | Lint `tests/project` with every rule and compare with `expected.txt` |
| `pnpm lint`                    | Lint this repo with the rules it passes; rebuilds the backend        |
| `pnpm format`                  | Format TS, docs, and Go                                              |
| `pnpm check`                   | Types, lint, formatting, Go/TS tests, and package integration        |
| `pnpm build`                   | Build JS, declarations, and the host binary                          |
| `pnpm pack`                    | Build all six native targets and create a tarball; does not publish  |
| `pnpm package:smoke <dir>`     | Install the tarball in `<dir>` and run both plugins                  |

## Adding a rule

1. Check for an existing Oxlint rule. Define one general problem, not a list of special cases.
2. Run `pnpm rule:new <name>`, with `--typed` for a type-aware rule. It creates `rules/<name>/` and registers the rule in `src/rules.ts`.
3. Replace the generated placeholders with a short rule contract, real valid/invalid tests, and an implementation.
4. Add the rule to the standalone CLI's defaults in `src/cli.ts`, type-aware rules with `--typed`, and to the README's rule table and default limits. The generator does neither.
5. Run focused tests, then `pnpm format` and `pnpm check`.

Every rule is tested by its folder in `tests/project/`: one TypeScript project that `pnpm test:project` lints with all rules in a single Oxlint run, comparing the findings with `tests/project/expected.txt`. `tests/project/oxlint.json` enables each rule only for its own folder. Generators handle registration and create the folder and its override. See [AGENTS.md](../AGENTS.md) for constraints and [native.md](native.md) for backend details.

## Package contents

Ship only `dist/`, rule `README.md` files, README, license, and package metadata. `dist/native/` includes six binaries, their rule manifests, source pins, and third-party notices. No tests, contributor guides, caches, or development scripts belong in the tarball.

The `aexlint` executable uses the packaged, pinned Oxlint dependency and standalone rule defaults. Build before running `node dist/src/cli.js check` or `node --test tests/cli.test.ts`. CLI tests use real temporary projects without installing tools into them and verify that project files remain unchanged.

Package tests resolve and fetch dependencies in a temporary consumer, then install offline from its lockfile and exercise the installed CLI executable. CI runs the full suite on Linux x64 and smoke-tests both the plugin and CLI from the same release tarball on Linux, macOS, and Windows, each on x64 and arm64.

## Releasing

Publish `@aetherall/aexlint` with public access. Review the contents and notices, and use the exact CI-tested tarball, not a local rebuild.

**First release:** set the version, commit and push without a tag, and wait for all CI jobs. Download the `package` artifact and publish its tarball with an authorized npm account. Then configure npm trusted publishing:

- Repository: `Aetherall/aexlint`
- Workflow: `release.yml`
- Environment: `npm`

After verifying OIDC publishing, select **Require two-factor authentication and disallow tokens** in npm's package settings. Keep account 2FA enabled; do not store npm tokens in GitHub secrets.

**Later releases:** update the version, commit, and push a matching `v<version>` tag. The release workflow checks the version, reruns CI, and publishes with provenance after environment approval. Never reuse a published version.

The GitHub `npm` environment permits only `v*` tags, requires approval from `Aetherall`, and disables administrator bypass. Self-approval is allowed; this is not two-person review.
