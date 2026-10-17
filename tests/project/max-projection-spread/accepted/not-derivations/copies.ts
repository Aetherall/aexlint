import type { Site } from "./site.js";
export function copies(site: Site, id: string, locale: string) {
  return { id, copy: site.id, label: site.getName().padEnd(locale.length), other: String(site.getName()) };
}
export function dates(date: Date) { return { at: date.toISOString(), day: date.getDay() }; }
export class Owner { private readonly name = ""; describe() { return { upper: this.name.toUpperCase(), lower: this.name.toLowerCase() }; } }
