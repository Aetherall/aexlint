export type Kind = "sundog" | "halo" | "corona";
export interface Item { kind: Kind; label: string; enabled: boolean }
export interface File { type: "breeze" } export interface Folder { type: "calm" | "doldrums" } export type Entity = File | Folder;
