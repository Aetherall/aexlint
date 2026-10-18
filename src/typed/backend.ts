import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export function backendExecutable(platform: string = process.platform): string {
  return platform === "win32" ? "aexlint-typed.exe" : "aexlint-typed";
}

export function defaultBackend(): string {
  return fileURLToPath(
    new URL(
      `../../native/${process.platform}-${process.arch}/${backendExecutable()}`,
      import.meta.url,
    ),
  );
}

export function ruleManifest(backend = defaultBackend()): string[] | null {
  const manifest = join(dirname(backend), "rules.json");
  if (!existsSync(backend) || !existsSync(manifest)) return null;
  let names: unknown;
  try {
    names = JSON.parse(readFileSync(manifest, "utf8"));
  } catch (error) {
    throw new Error(`The typed backend's rules.json is invalid: ${(error as Error).message}`, {
      cause: error,
    });
  }
  if (!Array.isArray(names) || !names.every((name) => typeof name === "string")) {
    throw new Error("The typed backend's rules.json is invalid: expected a list of rule names.");
  }
  return names;
}

function object(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object.`);
  }
  return value as Record<string, unknown>;
}

export interface NativeDiagnostic {
  kind: number;
  file_path: string | null;
  rule?: string;
  range?: { pos: number; end: number };
  message: { id?: string; description: string; help?: string };
}

export interface NativeMessages {
  diagnostics: NativeDiagnostic[];
  programFiles: string[];
  unmatchedFiles: string[];
}

export function decodeFrames(input: Uint8Array): NativeMessages {
  const bytes = Buffer.from(input.buffer, input.byteOffset, input.byteLength);
  const messages: NativeMessages = { diagnostics: [], programFiles: [], unmatchedFiles: [] };
  let offset = 0;
  while (offset < bytes.length) {
    if (bytes.length - offset < 5) throw new Error("Truncated native message header.");
    const size = bytes.readUInt32LE(offset);
    const type = bytes[offset + 4];
    offset += 5;
    if (size > bytes.length - offset) throw new Error("Truncated native message payload.");
    const payload: unknown = JSON.parse(bytes.subarray(offset, offset + size).toString("utf8"));
    offset += size;
    switch (type) {
      case 0:
        throw new Error(String(object(payload, "error").error));
      case 1:
        messages.diagnostics.push(nativeDiagnostic(payload));
        break;
      case 2:
        break;
      case 3:
        messages.programFiles.push(...programFiles(payload));
        break;
      case 4:
        messages.unmatchedFiles.push(...programFiles(payload));
        break;
      default:
        throw new Error(`Unknown native message type: ${type}`);
    }
  }
  return messages;
}

function nativeDiagnostic(payload: unknown): NativeDiagnostic {
  const value = object(payload, "diagnostic");
  const message = object(value.message, "diagnostic message");
  const located = typeof value.file_path === "string" || value.file_path === null;
  const valid = typeof value.kind === "number" && typeof message.description === "string";
  if (!located || !valid) throw new Error("Invalid native diagnostic.");
  return value as unknown as NativeDiagnostic;
}

function programFiles(payload: unknown): string[] {
  const files = object(payload, "program files").files;
  if (!Array.isArray(files) || !files.every((file) => typeof file === "string")) {
    throw new Error("Invalid native program files.");
  }
  return files;
}
