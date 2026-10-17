type Payload = { model: string };
const cache = new Map<string, Payload>();
export function get(key: string): Payload { const found = cache.get(key); if (!found || !found.model) throw new Error("x"); return found; }
