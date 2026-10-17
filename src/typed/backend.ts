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
  const names: unknown = JSON.parse(readFileSync(manifest, "utf8"));
  if (!Array.isArray(names) || !names.every((name) => typeof name === "string")) {
    throw new Error("Invalid native rule manifest.");
  }
  return names;
}

function object(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object.`);
  }
  return value as Record<string, unknown>;
}

export interface NativeMessages {
  diagnostics: unknown[];
  programFiles: string[];
}

export function decodeFrames(input: Uint8Array): NativeMessages {
  const bytes = Buffer.from(input.buffer, input.byteOffset, input.byteLength);
  const messages: NativeMessages = { diagnostics: [], programFiles: [] };
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
        messages.diagnostics.push(payload);
        break;
      case 2:
        break;
      case 3:
        messages.programFiles.push(...programFiles(payload));
        break;
      default:
        throw new Error(`Unknown native message type: ${type}`);
    }
  }
  return messages;
}

function programFiles(payload: unknown): string[] {
  const files = object(payload, "program files").files;
  if (!Array.isArray(files) || !files.every((file) => typeof file === "string")) {
    throw new Error("Invalid native program files.");
  }
  return files;
}
