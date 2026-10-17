import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { backendExecutable } from "../src/typed/backend.ts";
import { native } from "./native.ts";

const root = new URL("../", import.meta.url);
const oxlint = fileURLToPath(new URL("./bin/oxlint", import.meta.resolve("oxlint/package.json")));

try {
  await native("build");
  const target = `${process.platform}-${process.arch}`;
  const backend = fileURLToPath(new URL(`dist/native/${target}/${backendExecutable()}`, root));
  const result = spawnSync(
    process.execPath,
    [oxlint, "--deny-warnings", "src", "rules", "scripts", "tests"],
    { cwd: root, env: { ...process.env, AEXLINT_TYPED_BACKEND: backend }, stdio: "inherit" },
  );
  if (result.error) throw result.error;
  process.exitCode = result.status ?? 1;
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 2;
}
