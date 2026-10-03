import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { basename, join } from "node:path";
import { fileURLToPath } from "node:url";

export const root = fileURLToPath(new URL("../", import.meta.url));
const cache = join(root, ".native");
const source = join(cache, "source");
const pinText = readFileSync(join(root, "native/upstream.json"), "utf8");
const pins = JSON.parse(pinText) as Record<
  string,
  { repository: string; revision: string; sha256: string }
>;
const localPatches = readdirSync(join(root, "native/patches"))
  .filter((name) => name.endsWith(".patch"))
  .toSorted();
const stamp = createHash("sha256")
  .update(
    pinText +
      localPatches
        .map((name) => readFileSync(join(root, "native/patches", name), "utf8"))
        .join("\n"),
  )
  .digest("hex");
export const releaseTargets = [
  "linux-x64",
  "linux-arm64",
  "darwin-x64",
  "darwin-arm64",
  "win32-x64",
  "win32-arm64",
];
const goEnv = {
  ...process.env,
  CGO_ENABLED: "0",
  GOTOOLCHAIN: "local",
  GOWORK: join(source, "go.work"),
};

function run(
  command: string,
  args: string[],
  cwd = source,
  env: NodeJS.ProcessEnv = goEnv,
): string {
  const result = spawnSync(command, args, {
    cwd,
    env,
    encoding: "utf8",
    maxBuffer: 32 * 1024 * 1024,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")}\n${result.stdout}${result.stderr}`);
  }
  if (result.stderr) process.stderr.write(result.stderr);
  return result.stdout;
}

export function ruleNames(directory = join(root, "native/rules")): string[] {
  return readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .toSorted();
}

async function prepare(): Promise<void> {
  const readyFile = join(source, ".aexlint-ready");
  if (existsSync(readyFile) && readFileSync(readyFile, "utf8") === stamp) return;
  mkdirSync(cache, { recursive: true });
  rmSync(source, { recursive: true, force: true });
  mkdirSync(source);
  for (const [name, pin] of Object.entries(pins)) {
    const archive = join(cache, `${name}-${pin.revision}.tar.gz`);
    if (!existsSync(archive)) {
      console.log(`Fetching pinned ${name} ${pin.revision}`);
      const response = await fetch(
        `https://codeload.github.com/${pin.repository}/tar.gz/${pin.revision}`,
      );
      if (!response.ok) throw new Error(`Download ${name}: HTTP ${response.status}`);
      const bytes = Buffer.from(await response.arrayBuffer());
      if (createHash("sha256").update(bytes).digest("hex") !== pin.sha256) {
        throw new Error(`${name}: source checksum mismatch`);
      }
      writeFileSync(archive, bytes);
    }
    if (createHash("sha256").update(readFileSync(archive)).digest("hex") !== pin.sha256) {
      throw new Error(`${name}: cached source checksum mismatch`);
    }
    const destination = name === "tsgolint" ? source : join(source, "typescript-go");
    mkdirSync(destination, { recursive: true });
    run("tar", ["-xzf", archive, "--strip-components=1", "-C", destination], root);
  }
  for (const patch of readdirSync(join(source, "patches"))
    .filter((name) => name.endsWith(".patch"))
    .toSorted()) {
    run(
      "patch",
      ["-p1", "--batch", "--forward", "-i", join(source, "patches", patch)],
      join(source, "typescript-go"),
    );
  }
  for (const patch of localPatches) {
    run("patch", ["-p1", "--batch", "--forward", "-i", join(root, "native/patches", patch)]);
  }
  const collections = join(source, "internal/collections");
  mkdirSync(collections, { recursive: true });
  for (const file of readdirSync(join(source, "typescript-go/internal/collections"))) {
    if (file.endsWith(".go") && !file.endsWith("_test.go")) {
      cpSync(join(source, "typescript-go/internal/collections", file), join(collections, file));
    }
  }
  writeFileSync(join(source, ".aexlint-ready"), stamp);
}

function overlay(probe: boolean): string[] {
  const destination = join(source, "internal/aexlint");
  rmSync(destination, { recursive: true, force: true });
  cpSync(join(root, "native/rules"), join(destination, "rules"), { recursive: true });
  cpSync(join(root, "native/fixtures"), join(destination, "fixtures"), { recursive: true });
  if (probe) cpSync(join(root, "native/probe"), join(destination, "rules"), { recursive: true });
  const names = ruleNames(join(destination, "rules"));
  const imports = names.map(
    (name, index) =>
      `r${index} "github.com/typescript-eslint/tsgolint/internal/aexlint/rules/${name}"`,
  );
  const entries = names.map((_, index) => `r${index}.Rule`);
  writeFileSync(
    join(source, "cmd/tsgolint/zz_aexlint.go"),
    `package main\nimport (\n"github.com/typescript-eslint/tsgolint/internal/rule"\n${imports.join("\n")}\n)\nfunc init() {\nallRules = []rule.Rule{${entries.join(",")}}\nallRulesByName = make(map[string]rule.Rule, len(allRules))\nfor _, r := range allRules { allRulesByName[r.Name] = r }\n}\n`,
  );
  return names;
}

function assertReleaseTarget(target: string): void {
  if (!releaseTargets.includes(target)) throw new Error(`Unsupported native target: ${target}`);
}

function build(target: string, names: string[], directory: string): void {
  assertReleaseTarget(target);
  const [platform, arch] = target.split("-");
  const output = join(directory, target);
  mkdirSync(output, { recursive: true });
  const env = {
    ...goEnv,
    GOOS: platform === "win32" ? "windows" : platform!,
    GOARCH: arch === "x64" ? "amd64" : arch!,
  };
  console.log(`Building native ${target}`);
  run(
    "go",
    [
      "build",
      "-mod=readonly",
      "-buildvcs=false",
      "-trimpath",
      "-ldflags=-s -w",
      "-o",
      join(output, `aexlint-typed${platform === "win32" ? ".exe" : ""}`),
      "./cmd/tsgolint",
    ],
    source,
    env,
  );
  writeFileSync(
    join(output, "rules.json"),
    JSON.stringify(
      names.map((name) => `aexlint/${name}`),
      null,
      2,
    ) + "\n",
  );
}

function notices(directory: string): void {
  const modules = new Set(
    run("go", [
      "list",
      "-mod=readonly",
      "-deps",
      "-f",
      "{{with .Module}}{{.Dir}}{{end}}",
      "./cmd/tsgolint",
    ])
      .trim()
      .split("\n")
      .filter(Boolean),
  );
  modules.add(run("go", ["env", "GOROOT"]).trim());
  let text = `Native backend sources (modified with aexlint rules):\n${pinText}\n`;
  for (const module of [...modules].toSorted()) {
    for (const file of readdirSync(module)
      .filter((name) => /^(LICENSE|LICENCE|COPYING|NOTICE)(\.|$)/i.test(name))
      .toSorted()) {
      text += `\n--- ${basename(module)}/${file} ---\n${readFileSync(join(module, file), "utf8")}\n`;
    }
  }
  writeFileSync(join(directory, "THIRD_PARTY_NOTICES.txt"), text);
  cpSync(join(root, "native/upstream.json"), join(directory, "upstream.json"));
}

function snapshots(): Map<string, string> {
  const directory = join(source, "internal/rule_tester/__snapshots__/aexlint");
  return new Map(
    existsSync(directory)
      ? readdirSync(directory).map((file) => [file, readFileSync(join(directory, file), "utf8")])
      : [],
  );
}

function isFormattingAction(action: string): boolean {
  return ["format", "format:check"].includes(action);
}

function assertSourceAction(action: string): void {
  if (!["build", "release", "test", "prepare"].includes(action)) {
    throw new Error(`Unknown native action: ${action}`);
  }
}

export async function native(action: string, args: string[] = []): Promise<void> {
  if (isFormattingAction(action)) {
    const output = run("gofmt", [action === "format" ? "-w" : "-l", "native"], root);
    if (action === "format:check" && output.trim()) {
      throw new Error(`Run pnpm native:format:\n${output}`);
    }
    return;
  }
  assertSourceAction(action);
  await prepare();
  if (action === "prepare") return;
  const names = overlay(action === "test");
  if (action === "test") {
    const update = args.includes("--update");
    const filter = args.filter((arg) => arg !== "--update");
    const snapshotDir = join(source, "internal/rule_tester/__snapshots__/aexlint");
    rmSync(snapshotDir, { recursive: true, force: true });
    cpSync(join(root, "native/snapshots"), snapshotDir, { recursive: true });
    const before = snapshots();
    const packages = filter.length
      ? filter.map((name) => {
          if (!names.includes(name)) throw new Error(`Unknown native rule: ${name}`);
          return `./internal/aexlint/rules/${name}`;
        })
      : ["./internal/aexlint/rules/..."];
    process.stdout.write(
      run("go", ["test", "-mod=readonly", "-count=1", ...packages], source, {
        ...goEnv,
        UPDATE_SNAPS: update ? "true" : "false",
      }),
    );
    if (update) cpSync(snapshotDir, join(root, "native/snapshots"), { recursive: true });
    else {
      const actualSnapshots = JSON.stringify([...snapshots()].toSorted());
      const expectedSnapshots = JSON.stringify([...before].toSorted());
      if (actualSnapshots !== expectedSnapshots) {
        throw new Error(
          "Native snapshots changed. Review with pnpm native:test --update and retain the snapshots.",
        );
      }
    }
    build(`${process.platform}-${process.arch}`, names, join(cache, "probe"));
  } else {
    const directory = join(root, "dist/native");
    const targets = action === "release" ? releaseTargets : [`${process.platform}-${process.arch}`];
    for (const target of targets) build(target, names, directory);
    notices(directory);
  }
}

if (import.meta.main) {
  try {
    await native(process.argv[2] ?? "build", process.argv.slice(3));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
