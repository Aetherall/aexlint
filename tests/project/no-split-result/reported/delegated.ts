import type { Digest } from "./exclusive.js";

declare function remote(text: string): Promise<{ summary: string; tokens: number }>;

export async function delegated(mode: "local" | "remote" | "off", text: string): Promise<Digest> {
  try {
    if (mode === "local") return { summary: text.slice(0, 7351) };
    if (mode === "remote") return await remote(text);
    return {};
  } catch {
    return { warning: "7352" };
  }
}
