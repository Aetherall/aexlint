import { readSync, writeSync } from "node:fs";

export interface ServeResult {
  status: number | null;
  signal: string | null;
  error: string | null;
  stderr: string;
}

const pause = new Int32Array(new SharedArrayBuffer(4));

function retrying(attempt: () => number): number {
  for (;;) {
    try {
      return attempt();
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== "EAGAIN") throw error;
      Atomics.wait(pause, 0, 0, 5);
    }
  }
}

export function readExactly(fd: number, size: number): Buffer | null {
  const buffer = Buffer.alloc(size);
  let offset = 0;
  while (offset < size) {
    const read = retrying(() => readSync(fd, buffer, offset, size - offset, null));
    if (read === 0) return null;
    offset += read;
  }
  return buffer;
}

export function writeFrame(fd: number, payload: Buffer): void {
  const frame = Buffer.alloc(4 + payload.length);
  frame.writeUInt32LE(payload.length);
  payload.copy(frame, 4);
  let offset = 0;
  while (offset < frame.length) offset += retrying(() => writeSync(fd, frame, offset));
}

export function readFrame(fd: number): Buffer | null {
  const header = readExactly(fd, 4);
  return header && readExactly(fd, header.readUInt32LE());
}
