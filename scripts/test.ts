import { spawnSync } from "node:child_process";
import { globSync } from "node:fs";

const files = globSync(["tests/*.test.ts", "src/**/*.test.ts"]).toSorted();
if (!files.length) throw new Error("No tests discovered.");
const result = spawnSync(process.execPath, ["--test", ...files], { stdio: "inherit" });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
