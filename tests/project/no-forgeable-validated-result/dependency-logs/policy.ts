export type Tool = { name: string; run(): void };
export function assertTool(tool: Tool): void { if (!tool.name) console.warn("unnamed tool"); }
