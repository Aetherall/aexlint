import { readSync, writeSync } from "node:fs";

export interface ServeResult {
  status: number | null;
  signal: string | null;
  error: string | null;
  stderr: string;
}

export function readExactly(fd: number, size: number): Buffer | null {
  const buffer = Buffer.alloc(size);
  let offset = 0;
  while (offset < size) {
    const read = readSync(fd, buffer, offset, size - offset, null);
    if (read === 0) return null;
    offset += read;
  }
  return buffer;
}

export function writeFrame(fd: number, payload: Buffer): void {
  const header = Buffer.alloc(4);
  header.writeUInt32LE(payload.length);
  writeSync(fd, header);
  writeSync(fd, payload);
}

export function readFrame(fd: number): Buffer | null {
  const header = readExactly(fd, 4);
  return header && readExactly(fd, header.readUInt32LE());
}
