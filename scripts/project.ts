import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { backendExecutable } from "../src/typed/backend.ts";
import { native, root } from "./native.ts";

const project = join(root, "tests/project");
const expectedFile = join(project, "expected.txt");
const oxlint = fileURLToPath(new URL("./bin/oxlint", import.meta.resolve("oxlint/package.json")));

interface Diagnostic {
  code: string;
  message: string;
  help?: string;
  filename: string;
  labels: { span: { offset: number; length: number; line: number; column: number } }[];
}

function describe({ code, message, help, filename, labels }: Diagnostic): string {
  const path = filename.replaceAll("\\", "/");
  const [label] = labels;
  if (!label) return [`${path} ${code}`, `  ${message}`].join("\n");
  const { offset, length, line, column } = label.span;
  const source = readFileSync(join(project, filename));
  const text = source
    .subarray(offset, offset + length)
    .toString("utf8")
    .replaceAll("\n", "⏎");
  const lines = [
    `${path}:${line}:${column} ${code}`,
    `  ~ ${text}`,
    `  ${message.replaceAll("\n", "\n  ")}`,
  ];
  if (help) lines.push(`  help: ${help}`);
  return lines.join("\n");
}

function byPosition(a: Diagnostic, b: Diagnostic): number {
  const files = a.filename.localeCompare(b.filename);
  if (files) return files;
  return (a.labels[0]?.span.offset ?? -1) - (b.labels[0]?.span.offset ?? -1);
}

export function lintProject(): string {
  const target = `${process.platform}-${process.arch}`;
  const backend = join(root, "dist/native", target, backendExecutable());
  const { NODE_TEST_CONTEXT: _, ...env } = process.env;
  const result = spawnSync(
    process.execPath,
    [oxlint, "-c", "oxlint.json", "--format", "json", "."],
    {
      cwd: project,
      env: { ...env, AEXLINT_TYPED_BACKEND: backend },
      encoding: "utf8",
      maxBuffer: 256 * 1024 * 1024,
    },
  );
  if (result.error) throw result.error;
  if (result.status !== 0 && result.status !== 1) {
    throw new Error(`oxlint exited with ${result.status}\n${result.stderr}${result.stdout}`);
  }
  const { diagnostics } = JSON.parse(result.stdout) as { diagnostics: Diagnostic[] };
  const sorted = diagnostics.toSorted(byPosition);
  return sorted.map(describe).join("\n\n") + "\n";
}

export function addProjectRule(repository: string, name: string, rule: string): void {
  const folder = join(repository, "tests/project", name);
  const configFile = join(repository, "tests/project/oxlint.json");
  mkdirSync(folder, { recursive: true });
  writeFileSync(join(folder, "reported.ts"), "export const value = 2;\n", { flag: "wx" });
  const config = existsSync(configFile)
    ? JSON.parse(readFileSync(configFile, "utf8"))
    : { categories: { correctness: "off" }, overrides: [] };
  config.overrides.push({ files: [`${name}/**`], rules: { [rule]: "error" } });
  writeFileSync(configFile, JSON.stringify(config, null, 2) + "\n");
}

export function expectedProject(): string {
  return readFileSync(expectedFile, "utf8");
}

function blocks(text: string): Set<string> {
  return new Set(text.split("\n\n").map((block) => block.trim()));
}

function mark(sign: string, block: string): string {
  return block
    .split("\n")
    .map((line) => `${sign} ${line}`)
    .join("\n");
}

export function differences(expected: string, actual: string): string {
  const before = blocks(expected);
  const after = blocks(actual);
  const removed = [...before].filter((block) => !after.has(block));
  const added = [...after].filter((block) => !before.has(block));
  const marked = [
    ...removed.map((block) => mark("-", block)),
    ...added.map((block) => mark("+", block)),
  ];
  return marked.join("\n\n");
}

if (import.meta.main) {
  const update = process.argv.includes("--update");
  try {
    await native("build");
    const actual = lintProject();
    if (update) {
      writeFileSync(expectedFile, actual);
      console.log(`Updated ${expectedFile}`);
    } else {
      const diff = differences(expectedProject(), actual);
      if (diff) {
        console.log(diff);
        console.log("\nProject findings changed. Review, then run with --update.");
        process.exitCode = 1;
      } else {
        console.log("Project findings match.");
      }
    }
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  }
}
