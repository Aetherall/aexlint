export type Manifest = { code: string; parcels: string[] };
declare function log(value: unknown): void;
export function values(value: Manifest | undefined): string[] {
  if (value === undefined) return [];
  return value.parcels;
}
export function label(value: Manifest | undefined): string {
  if (!value) return "unlisted";
  return value.code;
}
export function create(value: Manifest | undefined): Manifest {
  if (!value) return { code: "pending", parcels: [] };
  return value;
}
export function logged(value: Manifest | undefined): string[] {
  log(value);
  if (!value) return [];
  return value.parcels;
}
export function nonEmpty(value: Manifest | undefined): string[] {
  if (!value || value.parcels.length === 0) return [];
  return value.parcels;
}
export function pair(code: string, value: Manifest | undefined) {
  if (!value) return undefined;
  return code + value.code;
}
export function only(value: Manifest | null | undefined): string[] {
  if (value === undefined) return [];
  return value ? value.parcels : [];
}
export function upper(text: string | undefined) {
  if (text === undefined) return undefined;
  return text.toUpperCase();
}
export function withDefault(value: Manifest | undefined = { code: "", parcels: [] }): string[] {
  if (!value) return [];
  return value.parcels;
}
export function many(...list: (Manifest | undefined)[]): string[] {
  if (!list[0]) return [];
  return list[0].parcels;
}
export function show(value: unknown) {
  if (!value) return undefined;
  return String(value);
}
export function guarded(value: Manifest | undefined, express: boolean): string[] {
  return value && express ? value.parcels : [];
}
export function visit(value: Manifest | undefined) {
  if (value) {
    log(value.code);
  }
}
export const ready = (value: Manifest | undefined, express: boolean) => value !== undefined && express && value.parcels.length > 0;
export function bound(this: void, value: Manifest | undefined): string[] {
  if (!value) return [];
  return value.parcels;
}
export async function load(value: Manifest | undefined) {
  if (!value) return undefined;
  return value.parcels;
}
export function* each(value: Manifest | undefined) {
  if (!value) return undefined;
  yield* value.parcels;
}
export function fixed(value: Manifest | undefined): string[] {
  if (!value) return [];
  return ["registered"];
}
export function required(value: Manifest): string[] {
  if (!value) return [];
  return value.parcels;
}
export const firstParcel = (value: Manifest | undefined) => value && value.parcels[0];
export function touch(value: Manifest | undefined) {
  value && log(value.code);
}
export declare function external(value: Manifest | undefined): string[];
export declare function wrap(list: string[]): string[];
export declare function now(): number;
export declare function stamped(value: Manifest | undefined, at: number): string[];
