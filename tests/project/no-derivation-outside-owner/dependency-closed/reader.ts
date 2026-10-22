import { gauge, type Spool } from "./spool.js";

export function wound(spool: Spool): string {
  return String(gauge(spool.thread));
}

export function measured(spool: Spool): string {
  return String(gauge(spool.thread));
}

export function cut(spool: Spool): string {
  return String(gauge(spool.thread));
}
