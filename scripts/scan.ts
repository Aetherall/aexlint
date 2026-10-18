import { spawnSync, type SpawnSyncOptionsWithBufferEncoding } from "node:child_process";
import { relative, resolve } from "node:path";
import { decodeFrames } from "../src/typed/backend.ts";
import { ruleNames } from "./native.ts";

const usage = "Usage: node scripts/scan.ts <facts|rules> <backend> <project root> <seed file>...";

function rules(mode: string): { name: string }[] {
  if (mode === "facts") return [{ name: "aexlint/model-facts" }];
  if (mode !== "rules") throw new Error(usage);
  return ruleNames().map((name) => ({ name: `aexlint/${name}` }));
}

const headless: SpawnSyncOptionsWithBufferEncoding = {
  maxBuffer: 1 << 30,
  env: { ...process.env, OXLINT_TSGOLINT_DANGEROUSLY_SUPPRESS_PROGRAM_DIAGNOSTICS: "true" },
};

function scan(mode: string, backend: string, root: string, seed: string): string[] {
  const configs = [{ file_paths: [resolve(root, seed)], rules: rules(mode) }];
  const input = JSON.stringify({ version: 2, whole_programs: true, configs });
  const result = spawnSync(backend, ["headless"], { ...headless, cwd: root, input });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${seed}: ${String(result.stderr).slice(0, 2000)}`);
  return decodeFrames(result.stdout).diagnostics.map((d) => {
    const text = `${d.message.description}\t${d.message.help ?? ""}`.replaceAll("\n", " ");
    const file = relative(root, d.file_path ?? "");
    return [seed, d.rule, file, d.range?.pos, d.range?.end, d.message.id, text].join("\t");
  });
}

const [mode, backend, root, ...seeds] = process.argv.slice(2);
if (!mode || !backend || !root || !seeds.length) throw new Error(usage);
const lines = seeds.flatMap((seed) => scan(mode, resolve(backend), resolve(root), seed)).toSorted();
process.stdout.write(lines.map((line) => `${line}\n`).join(""));
