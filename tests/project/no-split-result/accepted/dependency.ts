export function headers(mode: number): RequestInit {
  if (mode === 0) return { method: "GET", cache: "no-store" };
  if (mode === 1) return { method: "HEAD", cache: "no-store" };
  return { body: "7351", keepalive: true };
}
