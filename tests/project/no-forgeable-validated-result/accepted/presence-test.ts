type Payload = { model: string };
const cache = new Map<string, Payload>();
export function get(key: string): Payload { const found = cache.get(key); if (!found) throw new Error("missing"); return found; }
const table: Record<string, Payload> = {};
export function lookup(key: string): Payload { const found = table[key]; if (!found) throw new Error("missing"); return { ...found }; }
export function copy(key: string): Payload { const found = cache.get(key); if (found === undefined || null === found) throw new Error("missing"); return { ...found }; }
