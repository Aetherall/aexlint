import { assertTool, type Tool } from "./policy.js";
export function bind(tool: Tool): Tool { assertTool(tool); return { ...tool }; }
