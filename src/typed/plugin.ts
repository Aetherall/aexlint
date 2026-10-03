import { spawn, spawnSync } from "node:child_process";
import {
  closeSync,
  constants,
  existsSync,
  mkdtempSync,
  openSync,
  readFileSync,
  readSync,
  rmSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { definePlugin, defineRule, type Context, type Rule } from "@oxlint/plugins";
import { availableRules, decodeFrames, defaultBackend } from "./backend.ts";
import { readFrame, writeFrame, type ServeResult } from "./frames.ts";

interface NativeDiagnostic {
  kind: number;
  file_path: string | null;
  rule?: string;
  range?: { pos: number; end: number };
  message: { description: string; help?: string };
}

interface Request {
  name: string;
  options?: unknown;
}

interface Pending {
  file: string;
  requests: Request[];
  diagnostics: NativeDiagnostic[] | null;
}

interface BackendRun {
  result: ServeResult;
  stdout: Buffer;
}

type Channel = (input: Buffer) => BackendRun;

function withoutStack<T>(run: () => T): T {
  try {
    return run();
  } catch (error) {
    if (error instanceof Error && error.constructor === Error) {
      error.message = `aexlint-typed: ${error.message}`;
      error.stack = "";
    }
    throw error;
  }
}

const backend = process.env.AEXLINT_TYPED_BACKEND || defaultBackend();
const names = withoutStack(() => availableRules(backend));
const environment = {
  ...process.env,
  OXLINT_TSGOLINT_DANGEROUSLY_SUPPRESS_PROGRAM_DIAGNOSTICS: "true",
};

function directChannel(input: Buffer): BackendRun {
  const run = spawnSync(backend, ["headless"], {
    input,
    maxBuffer: 1024 * 1024 * 1024,
    env: environment,
  });
  const stderr = run.stderr ? run.stderr.toString("utf8") : "";
  const error = run.error ? run.error.message : null;
  return { result: { status: run.status, signal: run.signal, error, stderr }, stdout: run.stdout };
}

function logText(log: number): string {
  const buffer = Buffer.alloc(8192);
  const text = buffer.subarray(0, readSync(log, buffer, 0, buffer.length, 0)).toString("utf8");
  const panic = text.split("\n").find((line) => /^(?:panic|fatal error): /.test(line));
  return (panic ?? text)
    .replace(/^panic: /, "")
    .replace(" [recovered, repanicked]", "")
    .trim();
}

const startTimeout = 10_000;
const readyFrame = Buffer.from([5, 0, 0, 0, ...Buffer.from("ready")]);

const processTable = existsSync("/proc/self/stat");

function exited(pid: number | undefined): boolean {
  if (pid === undefined) return true;
  if (!processTable) return false;
  try {
    const stat = readFileSync(`/proc/${pid}/stat`, "utf8");
    return /^[ZX]/.test(stat.slice(stat.lastIndexOf(")") + 2));
  } catch {
    return true;
  }
}

function readReady(probe: number): boolean {
  const buffer = Buffer.alloc(readyFrame.length);
  try {
    const read = readSync(probe, buffer, 0, buffer.length, null);
    if (read === 0) return false;
    if (!buffer.subarray(0, read).equals(readyFrame)) {
      throw new Error("The typed backend server sent an unexpected greeting.");
    }
    return true;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "EAGAIN") return false;
    throw error;
  }
}

function awaitServer(path: string, pid: number | undefined, log: number): number {
  const probe = openSync(path, constants.O_RDONLY | constants.O_NONBLOCK);
  const pause = new Int32Array(new SharedArrayBuffer(4));
  const deadline = Date.now() + startTimeout;
  try {
    while (!readReady(probe)) {
      if (exited(pid) || Date.now() > deadline) {
        throw new Error(`The typed backend did not start: ${logText(log)}`);
      }
      Atomics.wait(pause, 0, 0, 5);
    }
    return openSync(path, "r");
  } finally {
    closeSync(probe);
  }
}

function serveChannel(): Channel {
  const directory = mkdtempSync(join(tmpdir(), "aexlint-typed-fifo-"));
  const requestPath = join(directory, "requests");
  const responsePath = join(directory, "responses");
  const fifo = spawnSync("mkfifo", [requestPath, responsePath]);
  if (fifo.status !== 0) throw new Error(`mkfifo failed: ${fifo.error?.message ?? fifo.stderr}`);
  const requests = openSync(requestPath, "r+");
  const log = openSync(join(directory, "stderr"), "w+");
  const server = spawn(backend, ["serve", requestPath, responsePath], {
    stdio: ["ignore", "ignore", log],
    env: environment,
  });
  server.on("error", () => {});
  server.unref();
  process.once("exit", () => rmSync(directory, { recursive: true, force: true }));
  let responses: number | null = null;
  let stopped = false;
  return (input) => {
    if (stopped) {
      throw new Error(
        "The typed backend stopped earlier in this run, so this file was not linted.",
      );
    }
    writeFrame(requests, input);
    if (responses === null) {
      try {
        responses = awaitServer(responsePath, server.pid, log);
      } catch (error) {
        stopped = true;
        server.kill();
        throw error;
      }
      rmSync(directory, { recursive: true, force: true });
    }
    const header = readFrame(responses);
    const stdout = readFrame(responses);
    if (!header || !stdout) {
      stopped = true;
      const output = logText(log);
      closeSync(log);
      throw new Error(`The typed backend stopped: ${output}`);
    }
    return { result: JSON.parse(header.toString("utf8")) as ServeResult, stdout };
  };
}

const channel: Channel = process.platform === "win32" ? directChannel : withoutStack(serveChannel);
const batches = new Map<string, Map<string, NativeDiagnostic[]>>();
let pending: Pending = { file: "", requests: [], diagnostics: null };

function request(context: Context, name: string): void {
  if (pending.file !== context.filename) {
    pending = { file: context.filename, requests: [], diagnostics: null };
  }
  const options: unknown = context.options[0];
  pending.requests.push(options === undefined ? { name } : { name, options });
}

function runBatch(
  file: string,
  text: string,
  requests: Request[],
): Map<string, NativeDiagnostic[]> {
  const onDisk = readFileSync(file, "utf8") === text;
  const payload = {
    version: 2,
    configs: [{ file_paths: [file], rules: requests }],
    whole_programs: true,
    ...(onDisk ? {} : { source_overrides: { [file]: text } }),
  };
  const { result, stdout } = channel(Buffer.from(JSON.stringify(payload)));
  if (result.error) throw new Error(result.error);
  const messages = decodeFrames(stdout);
  if (result.status !== 0) {
    throw new Error(`The typed backend exited ${result.status ?? result.signal}: ${result.stderr}`);
  }
  const diagnostics = messages.diagnostics as NativeDiagnostic[];
  if (!messages.programFiles.includes(file)) throw new Error(unbuiltProgram(file, diagnostics));
  const byFile = new Map<string, NativeDiagnostic[]>(messages.programFiles.map((f) => [f, []]));
  for (const diagnostic of diagnostics.filter((item) => item.kind === 0 && item.file_path)) {
    const target = diagnostic.file_path!;
    byFile.set(target, [...(byFile.get(target) ?? []), diagnostic]);
  }
  return byFile;
}

function unbuiltProgram(file: string, diagnostics: NativeDiagnostic[]): string {
  const reasons = diagnostics
    .filter((item) => item.kind === 1)
    .map((item) => {
      const { description, help } = item.message;
      return `${item.file_path ?? "TypeScript"}: ${help ? `${description}: ${help}` : description}`;
    });
  return [`No TypeScript program could be built for ${file}.`, ...reasons].join("\n");
}

function diagnosticsFor(context: Context): NativeDiagnostic[] {
  if (pending.diagnostics) return pending.diagnostics;
  const file = context.filename.replaceAll("\\", "/");
  const text = context.sourceCode.text;
  const key = JSON.stringify(pending.requests);
  const cached = batches.get(key)?.get(file);
  const fresh = cached && readFileSync(file, "utf8") === text;
  let found = fresh ? cached : undefined;
  if (!found) {
    const store = batches.get(key) ?? new Map<string, NativeDiagnostic[]>();
    for (const [path, diagnostics] of runBatch(file, text, pending.requests)) {
      store.set(path, diagnostics);
    }
    batches.set(key, store);
    found = store.get(file) ?? [];
  }
  pending.diagnostics = found;
  return found;
}

function textOffset(text: string, bytes: Buffer, pos: number): number {
  return bytes.length === text.length ? pos : bytes.subarray(0, pos).toString("utf8").length;
}

function report(context: Context, name: string, diagnostic: NativeDiagnostic): void {
  if (diagnostic.rule !== name) return;
  const text = context.sourceCode.text;
  const bytes = Buffer.from(text, "utf8");
  const range = diagnostic.range ?? { pos: 0, end: 0 };
  const { description, help } = diagnostic.message;
  context.report({
    message: help ? `${description}\nhelp: ${help}` : description,
    node: { range: [textOffset(text, bytes, range.pos), textOffset(text, bytes, range.end)] },
  });
}

function wrap(name: string): Rule {
  return defineRule({
    meta: {
      type: "suggestion",
      docs: { description: `Runs the native ${name} rule through aexlint's typed backend.` },
      schema: false,
    },
    create(context) {
      request(context, name);
      return {
        Program() {
          withoutStack(() => {
            for (const diagnostic of diagnosticsFor(context)) report(context, name, diagnostic);
          });
        },
      };
    },
  });
}

export default definePlugin({
  meta: { name: "aexlint-typed" },
  rules: Object.fromEntries(names.map((name) => [name.replace(/^aexlint\//, ""), wrap(name)])),
});
