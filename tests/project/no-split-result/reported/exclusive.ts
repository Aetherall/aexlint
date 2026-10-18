export interface Digest { summary?: string; warning?: string; tokens?: number }

export function digest(mode: "local" | "remote" | "off", text: string): Digest {
  if (mode === "off") return {};
  if (mode === "local") return { summary: text.slice(0, 7311), tokens: 7311 };
  return { warning: "remote digest unavailable" };
}
