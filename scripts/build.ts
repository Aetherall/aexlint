import { spawnSync } from "node:child_process";
import { rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { native } from "./native.ts";

const root = new URL("../", import.meta.url);
const compiler = new URL("./bin/tsc", import.meta.resolve("typescript/package.json"));
rmSync(new URL("dist/", root), { recursive: true, force: true });
const result = spawnSync(process.execPath, [fileURLToPath(compiler), "-p", "tsconfig.build.json"], {
  cwd: root,
  stdio: "inherit",
});
if (result.error) throw result.error;
if (result.status !== 0) process.exitCode = result.status ?? 1;
else await native(process.argv.includes("--release") ? "release" : "build");
