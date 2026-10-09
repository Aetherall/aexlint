import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { backendExecutable } from "../src/typed/backend.ts";
import { native } from "./native.ts";

const root = new URL("../", import.meta.url);
const oxlint = fileURLToPath(new URL("./bin/oxlint", import.meta.resolve("oxlint/package.json")));
const directories = ["src", "scripts", "tests"];

function lint(args: string[], env: NodeJS.ProcessEnv = process.env): number {
  const result = spawnSync(process.execPath, [oxlint, "--deny-warnings", ...args, ...directories], {
    cwd: root,
    env,
    stdio: "inherit",
  });
  if (result.error) throw result.error;
  return result.status ?? 1;
}

try {
  const syntax = lint([]);
  await native("build");
  const target = `${process.platform}-${process.arch}`;
  const backend = fileURLToPath(new URL(`dist/native/${target}/${backendExecutable()}`, root));
  const typed = lint(["-c", ".oxlintrc.typed.json"], {
    ...process.env,
    AEXLINT_TYPED_BACKEND: backend,
  });
  process.exitCode = Math.max(syntax, typed);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 2;
}
