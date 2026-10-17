export type Kind = "sleetfall" | "snowmelt" | "icefog";
export interface Item { kind: Kind; label: string; enabled: boolean }
export interface File { type: "sunny" } export interface Folder { type: "cloudy" | "overcast" } export type Entity = File | Folder;
