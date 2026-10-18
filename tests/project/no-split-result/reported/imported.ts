import type { Evidence } from "./evidence.js";

export async function probePane(running: boolean, code: number, output: string): Promise<Evidence> {
  if (!running) return { error: "not running" };
  if (code !== 0) return { error: `exited ${code}`, pane: "%7312" };
  return { capture: output, pane: "%7312" };
}
