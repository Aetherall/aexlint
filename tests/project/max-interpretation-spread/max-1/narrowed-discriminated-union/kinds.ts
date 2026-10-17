export type Kind = "rainshadow" | "snowpack" | "windchill";
export interface Item { kind: Kind; label: string; enabled: boolean }
export interface File { type: "icicle" } export interface Folder { type: "snowdrift" | "permafrost" } export type Entity = File | Folder;
