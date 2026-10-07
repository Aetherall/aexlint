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

| Command                                          | Purpose                                                             |
| ------------------------------------------------ | ------------------------------------------------------------------- |
| `pnpm test:rule src/rules/<name>/<name>.test.ts` | Test one JS rule; accepts `--watch`                                 |
| `pnpm native:test <name>`                        | Test one Go rule                                                    |
| `pnpm native:test <name> --update`               | Update snapshots; review the changes                                |
| `pnpm lint`                                      | Self-lint with both plugins; rebuilds the host backend              |
| `pnpm lint:classifications`                      | Optional experimental rule review                                   |
| `pnpm format`                                    | Format TS, docs, and Go                                             |
| `pnpm check`                                     | Types, lint, formatting, Go/TS tests, and package integration       |
| `pnpm build`                                     | Build JS, declarations, and the host binary                         |
| `pnpm pack`                                      | Build all six native targets and create a tarball; does not publish |
| `pnpm package:smoke <dir>`                       | Install the tarball in `<dir>` and run both plugins                 |

## Adding a rule

1. Check for an existing Oxlint rule. Define one general problem, not a list of special cases.
2. Run `pnpm rule:new <name>` for syntax, or `pnpm rule:new:typed <name>` for types.
3. Replace the generated placeholders with a short rule contract, real valid/invalid tests, and an implementation.
4. Run focused tests, then `pnpm format` and `pnpm check`.

Use Oxlint's real RuleTester for JS and the upstream Go tester for typed rules. Include cross-file type changes where relevant. Generators handle registration and intentionally start with failing tests. See [AGENTS.md](../AGENTS.md) for constraints and [native.md](native.md) for backend details.

## Package contents

Ship only `dist/`, rule docs, README, license, and package metadata. `dist/native/` includes six binaries, their rule manifests, source pins, and third-party notices. No tests, contributor guides, caches, or development scripts belong in the tarball.

Package tests resolve and fetch dependencies in a temporary consumer, then install offline from its lockfile. CI runs the full suite on Linux x64 and smoke-tests the same release tarball on Linux, macOS, and Windows, each on x64 and arm64.

## Releasing

Always review the package contents and notices. Publish the exact CI-tested tarball, not a local rebuild.

**First release:** set the version, commit and push without a tag, and wait for all CI jobs. Download the `package` artifact and publish its tarball with an authorized npm account. Then configure npm trusted publishing:

- Repository: `Aetherall/aexlint`
- Workflow: `release.yml`
- Environment: `npm`

After verifying OIDC publishing, select **Require two-factor authentication and disallow tokens** in npm's package settings. Keep account 2FA enabled; do not store npm tokens in GitHub secrets.

**Later releases:** update the version, commit, and push a matching `v<version>` tag. The release workflow checks the version, reruns CI, and publishes with provenance after environment approval. Never reuse a published version.

The GitHub `npm` environment permits only `v*` tags, requires approval from `Aetherall`, and disables administrator bypass. Self-approval is allowed; this is not two-person review.
